//go:build integration

package vectorstore

import (
	"context"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/google/uuid"
)

// qdrantAddr retorna o endereço do Qdrant para testes de integração.
// Dentro do Docker Compose, QDRANT_HOST=qdrant e QDRANT_PORT=6333.
func qdrantAddr(t *testing.T) string {
	t.Helper()
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("QDRANT_PORT")
	if port == "" {
		port = "6333"
	}
	return fmt.Sprintf("%s:%s", host, port)
}

// testCollectionName retorna um nome de collection único por teste
// para evitar interferência entre testes paralelos.
func testCollectionName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test_%s", t.Name())
}

// cleanupCollection remove a collection no final do teste.
func cleanupCollection(t *testing.T, store *QdrantStore, name string) {
	t.Helper()
	ctx := context.Background()
	url := fmt.Sprintf("%s/collections/%s", store.baseURL, name)
	resp, err := store.doRequest(ctx, "DELETE", url, nil)
	if err != nil {
		t.Logf("cleanup: falha ao deletar collection %s: %v", name, err)
		return
	}
	resp.Body.Close()
}

// TestIntegrationEnsureCollection verifica que a collection é criada com sucesso
// e que chamadas subsequentes são idempotentes.
func TestIntegrationEnsureCollection(t *testing.T) {
	store := NewQdrant(qdrantAddr(t))
	ctx := context.Background()
	collection := testCollectionName(t)
	defer cleanupCollection(t, store, collection)

	// Primeira chamada: cria
	if err := store.EnsureCollection(ctx, collection, 1536); err != nil {
		t.Fatalf("EnsureCollection falhou: %v", err)
	}

	// Segunda chamada: idempotente
	if err := store.EnsureCollection(ctx, collection, 1536); err != nil {
		t.Fatalf("EnsureCollection (idempotente) falhou: %v", err)
	}
}

// TestIntegrationUpsertAndSearch verifica o fluxo completo:
// criar collection → inserir pontos → buscar pontos similares.
func TestIntegrationUpsertAndSearch(t *testing.T) {
	store := NewQdrant(qdrantAddr(t))
	ctx := context.Background()
	collection := testCollectionName(t)
	defer cleanupCollection(t, store, collection)

	const dim = 4 // Dimensão pequena para testes rápidos

	if err := store.EnsureCollection(ctx, collection, dim); err != nil {
		t.Fatalf("EnsureCollection falhou: %v", err)
	}

	// IDs determinísticos (UUIDs válidos exigidos pelo Qdrant)
	id1 := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("chunk-001")).String()
	id2 := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("chunk-002")).String()
	id3 := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("chunk-003")).String()

	// Insere 3 pontos com vetores distintos
	points := []Point{
		{
			ID:     id1,
			Vector: []float32{1.0, 0.0, 0.0, 0.0},
			Payload: map[string]any{
				"document_id": "doc-abc",
				"chunk_index": 0,
				"text":        "Como resetar o cache do sistema",
				"file_name":   "runbook.md",
				"file_type":   "md",
			},
		},
		{
			ID:     id2,
			Vector: []float32{0.0, 1.0, 0.0, 0.0},
			Payload: map[string]any{
				"document_id": "doc-abc",
				"chunk_index": 1,
				"text":        "Procedimento de deploy em produção",
				"file_name":   "runbook.md",
				"file_type":   "md",
			},
		},
		{
			ID:     id3,
			Vector: []float32{0.0, 0.0, 1.0, 0.0},
			Payload: map[string]any{
				"document_id": "doc-xyz",
				"chunk_index": 0,
				"text":        "Política de backup diário",
				"file_name":   "backup-guide.txt",
				"file_type":   "txt",
			},
		},
	}

	if err := store.Upsert(ctx, collection, points); err != nil {
		t.Fatalf("Upsert falhou: %v", err)
	}

	// Busca com vetor mais próximo de chunk-001
	query := []float32{0.9, 0.1, 0.0, 0.0}
	results, err := store.Search(ctx, collection, query, 2)
	if err != nil {
		t.Fatalf("Search falhou: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Search retornou %d resultados, want 2", len(results))
	}

	// O primeiro resultado deve ser id1 (vetor mais similar)
	if results[0].ID != id1 {
		t.Errorf("Primeiro resultado ID = %s, want %s", results[0].ID, id1)
	}

	if results[0].Score < 0.5 {
		t.Errorf("Score do primeiro resultado = %f, esperado > 0.5", results[0].Score)
	}

	// Verifica que o payload contém o texto
	text, ok := results[0].Payload["text"]
	if !ok {
		t.Error("Payload não contém campo 'text'")
	}
	if text != "Como resetar o cache do sistema" {
		t.Errorf("text = %v, want 'Como resetar o cache do sistema'", text)
	}
}

// TestIntegrationDeleteByDocID verifica que DeleteByDocID remove apenas
// os pontos do documento especificado.
func TestIntegrationDeleteByDocID(t *testing.T) {
	store := NewQdrant(qdrantAddr(t))
	ctx := context.Background()
	collection := testCollectionName(t)
	defer cleanupCollection(t, store, collection)

	const dim = 4

	if err := store.EnsureCollection(ctx, collection, dim); err != nil {
		t.Fatalf("EnsureCollection falhou: %v", err)
	}

	idKeep := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("del-001")).String()
	idRem1 := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("del-002")).String()
	idRem2 := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("del-003")).String()

	// Insere pontos de 2 documentos diferentes
	points := []Point{
		{
			ID:      idKeep,
			Vector:  []float32{1.0, 0.0, 0.0, 0.0},
			Payload: map[string]any{"document_id": "doc-keep", "text": "manter este"},
		},
		{
			ID:      idRem1,
			Vector:  []float32{0.0, 1.0, 0.0, 0.0},
			Payload: map[string]any{"document_id": "doc-remove", "text": "remover este"},
		},
		{
			ID:      idRem2,
			Vector:  []float32{0.0, 0.0, 1.0, 0.0},
			Payload: map[string]any{"document_id": "doc-remove", "text": "remover este tb"},
		},
	}

	if err := store.Upsert(ctx, collection, points); err != nil {
		t.Fatalf("Upsert falhou: %v", err)
	}

	// Remove pontos do doc-remove
	if err := store.DeleteByDocID(ctx, collection, "doc-remove"); err != nil {
		t.Fatalf("DeleteByDocID falhou: %v", err)
	}

	// Busca genérica: deve retornar apenas o ponto de doc-keep
	query := []float32{0.5, 0.5, 0.5, 0.5}
	results, err := store.Search(ctx, collection, query, 10)
	if err != nil {
		t.Fatalf("Search após delete falhou: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search retornou %d resultados após delete, want 1", len(results))
	}

	if results[0].ID != idKeep {
		t.Errorf("Resultado remanescente ID = %s, want %s", results[0].ID, idKeep)
	}
}

// TestIntegrationUpsertOverwrite verifica que upsertar com o mesmo ID
// atualiza o ponto existente.
func TestIntegrationUpsertOverwrite(t *testing.T) {
	store := NewQdrant(qdrantAddr(t))
	ctx := context.Background()
	collection := testCollectionName(t)
	defer cleanupCollection(t, store, collection)

	const dim = 4

	store.EnsureCollection(ctx, collection, dim)

	overwriteID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("overwrite-001")).String()

	// Insere ponto inicial
	store.Upsert(ctx, collection, []Point{
		{
			ID:      overwriteID,
			Vector:  []float32{1.0, 0.0, 0.0, 0.0},
			Payload: map[string]any{"text": "versão 1"},
		},
	})

	// Sobrescreve com novo vetor e payload
	store.Upsert(ctx, collection, []Point{
		{
			ID:      overwriteID,
			Vector:  []float32{0.0, 0.0, 0.0, 1.0},
			Payload: map[string]any{"text": "versão 2"},
		},
	})

	// Busca com vetor alinhado à versão 2
	results, err := store.Search(ctx, collection, []float32{0.0, 0.0, 0.0, 1.0}, 1)
	if err != nil {
		t.Fatalf("Search falhou: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Search retornou %d resultados, want 1", len(results))
	}

	text := results[0].Payload["text"]
	if text != "versão 2" {
		t.Errorf("text = %v, want 'versão 2'", text)
	}

	// Score deve ser ~1.0 (vetor idêntico)
	if math.Abs(float64(results[0].Score)-1.0) > 0.01 {
		t.Errorf("Score = %f, esperado ~1.0", results[0].Score)
	}
}
