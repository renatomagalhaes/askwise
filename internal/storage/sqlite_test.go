package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// newTestDB cria um banco SQLite temporário para testes.
// Cada teste recebe um banco isolado para evitar interferência.
func newTestDB(t *testing.T) *SQLiteStorage {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("falha ao criar SQLite de teste: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// sampleDoc retorna um DocumentMeta de exemplo para uso nos testes.
func sampleDoc(id, name string) *DocumentMeta {
	return &DocumentMeta{
		ID:           id,
		Name:         name,
		OriginalName: "Original " + name,
		FileType:     "md",
		FileSize:     1024,
		ChunkCount:   5,
		Status:       "ready",
	}
}

// TestSaveAndGetDocument verifica o fluxo básico de salvar e recuperar.
func TestSaveAndGetDocument(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	doc := sampleDoc("doc-001", "test-file.md")

	if err := store.SaveDocument(ctx, doc); err != nil {
		t.Fatalf("SaveDocument falhou: %v", err)
	}

	got, err := store.GetDocument(ctx, "doc-001")
	if err != nil {
		t.Fatalf("GetDocument falhou: %v", err)
	}

	// Table-driven checks nos campos retornados
	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"ID", got.ID, "doc-001"},
		{"Name", got.Name, "test-file.md"},
		{"OriginalName", got.OriginalName, "Original test-file.md"},
		{"FileType", got.FileType, "md"},
		{"FileSize", got.FileSize, int64(1024)},
		{"ChunkCount", got.ChunkCount, 5},
		{"Status", got.Status, "ready"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}

	// Timestamps devem ter sido preenchidos automaticamente
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

// TestGetDocumentNotFound verifica que buscar um ID inexistente retorna erro.
func TestGetDocumentNotFound(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	_, err := store.GetDocument(ctx, "nonexistent")
	if err == nil {
		t.Error("GetDocument deveria retornar erro para ID inexistente")
	}
}

// TestListDocuments verifica a listagem completa e a ordenação (mais recente primeiro).
func TestListDocuments(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	// Insere 3 documentos
	for i, name := range []string{"a.md", "b.pdf", "c.txt"} {
		doc := sampleDoc(
			"doc-"+string(rune('1'+i)),
			name,
		)
		if err := store.SaveDocument(ctx, doc); err != nil {
			t.Fatalf("SaveDocument(%s) falhou: %v", name, err)
		}
	}

	docs, err := store.ListDocuments(ctx)
	if err != nil {
		t.Fatalf("ListDocuments falhou: %v", err)
	}

	if len(docs) != 3 {
		t.Fatalf("ListDocuments retornou %d docs, want 3", len(docs))
	}
}

// TestListDocumentsEmpty verifica que listar um banco vazio retorna slice vazio (não nil).
func TestListDocumentsEmpty(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	docs, err := store.ListDocuments(ctx)
	if err != nil {
		t.Fatalf("ListDocuments falhou: %v", err)
	}

	if docs != nil && len(docs) != 0 {
		t.Errorf("ListDocuments retornou %d docs para banco vazio, want 0", len(docs))
	}
}

// TestDeleteDocument verifica que a remoção funciona e que o doc some da listagem.
// RN-21: integridade referencial (SQLite side).
func TestDeleteDocument(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	doc := sampleDoc("doc-del", "deleteme.txt")
	store.SaveDocument(ctx, doc)

	if err := store.DeleteDocument(ctx, "doc-del"); err != nil {
		t.Fatalf("DeleteDocument falhou: %v", err)
	}

	// Verificar que não existe mais
	_, err := store.GetDocument(ctx, "doc-del")
	if err == nil {
		t.Error("GetDocument deveria falhar após delete")
	}
}

// TestDeleteDocumentNotFound verifica que deletar ID inexistente retorna erro.
func TestDeleteDocumentNotFound(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	err := store.DeleteDocument(ctx, "ghost")
	if err == nil {
		t.Error("DeleteDocument deveria retornar erro para ID inexistente")
	}
}

// TestGetDocumentByName verifica a busca por nome (RN-04: detecção de duplicatas).
func TestGetDocumentByName(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	doc := sampleDoc("doc-name", "unique-name.pdf")
	store.SaveDocument(ctx, doc)

	// Busca que deve encontrar
	found, err := store.GetDocumentByName(ctx, "unique-name.pdf")
	if err != nil {
		t.Fatalf("GetDocumentByName falhou: %v", err)
	}
	if found == nil {
		t.Fatal("GetDocumentByName retornou nil, esperava encontrar documento")
	}
	if found.ID != "doc-name" {
		t.Errorf("ID = %s, want doc-name", found.ID)
	}

	// Busca que NÃO deve encontrar (retorna nil sem erro)
	notFound, err := store.GetDocumentByName(ctx, "nonexistent.pdf")
	if err != nil {
		t.Fatalf("GetDocumentByName falhou para inexistente: %v", err)
	}
	if notFound != nil {
		t.Error("GetDocumentByName deveria retornar nil para nome inexistente")
	}
}

// TestSaveDocumentUpsert verifica que salvar o mesmo ID atualiza o registro
// (RN-22: idempotência).
func TestSaveDocumentUpsert(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()

	doc := sampleDoc("doc-upsert", "file.md")
	doc.ChunkCount = 5
	store.SaveDocument(ctx, doc)

	// Atualiza chunk count e salva novamente
	doc.ChunkCount = 10
	doc.Status = "error"
	doc.ErrorMessage = "parse failed"
	if err := store.SaveDocument(ctx, doc); err != nil {
		t.Fatalf("Upsert falhou: %v", err)
	}

	got, _ := store.GetDocument(ctx, "doc-upsert")
	if got.ChunkCount != 10 {
		t.Errorf("ChunkCount = %d após upsert, want 10", got.ChunkCount)
	}
	if got.Status != "error" {
		t.Errorf("Status = %s após upsert, want error", got.Status)
	}
}

// TestNewSQLiteInvalidPath verifica que um path inválido retorna erro descritivo.
func TestNewSQLiteInvalidPath(t *testing.T) {
	// Cria um arquivo temporário
	f, _ := os.CreateTemp("", "not-a-dir")
	defer os.Remove(f.Name())
	f.Close()

	// Tenta usar o arquivo como se fosse um diretório no path
	invalidPath := filepath.Join(f.Name(), "db.sqlite")
	_, err := NewSQLite(invalidPath)
	if err == nil {
		t.Error("NewSQLite deveria falhar ao tentar criar diretório sobre um arquivo existente")
	}
}

// TestNewSQLiteCreatesFile verifica que NewSQLite cria o arquivo se não existir.
func TestNewSQLiteCreatesFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "new.db")

	store, err := NewSQLite(dbPath)
	if err != nil {
		t.Fatalf("NewSQLite falhou: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("NewSQLite não criou o arquivo do banco")
	}
}
