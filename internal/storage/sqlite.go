package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	// modernc.org/sqlite é uma implementação pure Go do SQLite — não precisa de CGO.
	// ADR-003: Escolhida por compilar em qualquer plataforma sem dependências C.
	_ "modernc.org/sqlite"
)

// sqliteSchema é o DDL executado na inicialização para criar a tabela e índices.
// Spec: design/02-MODELO-DADOS.md §2 (SQLite — Tabela documents)
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS documents (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL,
    file_type     TEXT NOT NULL,
    file_size     INTEGER NOT NULL,
    chunk_count   INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'processing',
    error_message TEXT,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_documents_name ON documents(name);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
`

// SQLiteStorage implementa Storage usando SQLite como backend.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLite cria uma nova instância de SQLiteStorage.
// Abre a conexão com o arquivo especificado em dbPath e executa o schema
// de auto-migrate (cria tabela e índices se não existirem).
func NewSQLite(dbPath string) (*SQLiteStorage, error) {
	// Garante que o diretório pai existe. Se dbPath for apenas o nome do arquivo,
	// filepath.Dir retornará "." que MkdirAll tratará sem erro.
	// modernc.org/sqlite falha com "out of memory" se o diretório não existir.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("falha ao criar diretório para o banco em %s: %w", dir, err)
	}

	// database/sql usa o driver registrado por modernc.org/sqlite via blank import.
	// O DSN "file:<path>" abre (ou cria) o arquivo SQLite.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir SQLite em %s: %w", dbPath, err)
	}

	// Verifica se a conexão está funcional.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("falha ao conectar ao SQLite: %w", err)
	}

	// Executa o schema para criar tabela e índices (idempotente via IF NOT EXISTS).
	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("falha ao executar schema SQLite: %w", err)
	}

	slog.Info("sqlite initialized", "component", "storage", "path", dbPath)
	return &SQLiteStorage{db: db}, nil
}

// SaveDocument insere ou atualiza os metadados de um documento.
// Usa INSERT OR REPLACE que funciona como upsert no SQLite — se o ID já
// existir, substitui o registro (RN-22: idempotência de upload).
func (s *SQLiteStorage) SaveDocument(ctx context.Context, doc *DocumentMeta) error {
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now

	query := `
		INSERT OR REPLACE INTO documents
			(id, name, original_name, file_type, file_size, chunk_count, status, error_message, created_at, updated_at)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		doc.ID, doc.Name, doc.OriginalName, doc.FileType,
		doc.FileSize, doc.ChunkCount, doc.Status, doc.ErrorMessage,
		doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("falha ao salvar documento %s: %w", doc.ID, err)
	}
	return nil
}

// GetDocument retorna os metadados de um documento pelo ID.
// Retorna erro se não encontrar (sql.ErrNoRows é propagado com contexto).
func (s *SQLiteStorage) GetDocument(ctx context.Context, id string) (*DocumentMeta, error) {
	query := `
		SELECT id, name, original_name, file_type, file_size, chunk_count,
		       status, error_message, created_at, updated_at
		FROM documents WHERE id = ?
	`
	doc := &DocumentMeta{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.Name, &doc.OriginalName, &doc.FileType,
		&doc.FileSize, &doc.ChunkCount, &doc.Status, &doc.ErrorMessage,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("documento %s não encontrado", id)
	}
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar documento %s: %w", id, err)
	}
	return doc, nil
}

// ListDocuments retorna todos os documentos, ordenados por data de criação
// (mais recente primeiro). RF-08: Listagem de documentos.
func (s *SQLiteStorage) ListDocuments(ctx context.Context) ([]DocumentMeta, error) {
	query := `
		SELECT id, name, original_name, file_type, file_size, chunk_count,
		       status, error_message, created_at, updated_at
		FROM documents ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar documentos: %w", err)
	}
	defer rows.Close()

	var docs []DocumentMeta
	for rows.Next() {
		var doc DocumentMeta
		if err := rows.Scan(
			&doc.ID, &doc.Name, &doc.OriginalName, &doc.FileType,
			&doc.FileSize, &doc.ChunkCount, &doc.Status, &doc.ErrorMessage,
			&doc.CreatedAt, &doc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("falha ao ler documento da lista: %w", err)
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// DeleteDocument remove um documento pelo ID.
// RN-21: Apenas remove os metadados do SQLite. O caller deve também
// remover os chunks/embeddings do Qdrant para manter integridade referencial.
func (s *SQLiteStorage) DeleteDocument(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("falha ao deletar documento %s: %w", id, err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("documento %s não encontrado", id)
	}
	return nil
}

// GetDocumentByName busca um documento pelo nome sanitizado.
// RN-04: Usado para detectar duplicatas antes de um novo upload.
func (s *SQLiteStorage) GetDocumentByName(ctx context.Context, name string) (*DocumentMeta, error) {
	query := `
		SELECT id, name, original_name, file_type, file_size, chunk_count,
		       status, error_message, created_at, updated_at
		FROM documents WHERE name = ?
	`
	doc := &DocumentMeta{}
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&doc.ID, &doc.Name, &doc.OriginalName, &doc.FileType,
		&doc.FileSize, &doc.ChunkCount, &doc.Status, &doc.ErrorMessage,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Não encontrado não é erro — significa que não é duplicata
	}
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar documento por nome %s: %w", name, err)
	}
	return doc, nil
}

// Close fecha a conexão com o banco de dados SQLite.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
