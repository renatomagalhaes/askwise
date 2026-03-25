package storage

import (
	"context"
	"time"
)

// DocumentMeta representa os metadados de um documento armazenado no SQLite.
// Spec: design/02-MODELO-DADOS.md §4 (Document Metadado)
type DocumentMeta struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"original_name"`
	FileType     string    `json:"file_type"`
	FileSize     int64     `json:"file_size"`
	ChunkCount   int       `json:"chunk_count"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Storage define a interface para persistência de metadados de documentos.
// Spec: design/01-ARQUITETURA.md §2.10
//
// A interface permite trocar a implementação (ex: de SQLite para PostgreSQL)
// sem alterar o restante do código — basta implementar esta interface.
type Storage interface {
	// SaveDocument persiste os metadados de um novo documento.
	// Se um documento com mesmo ID já existir, atualiza os campos.
	SaveDocument(ctx context.Context, doc *DocumentMeta) error

	// GetDocument retorna os metadados de um documento pelo ID.
	// Retorna error se o documento não for encontrado.
	GetDocument(ctx context.Context, id string) (*DocumentMeta, error)

	// ListDocuments retorna todos os documentos cadastrados,
	// ordenados por data de criação (mais recente primeiro).
	// RF-08: Listagem de documentos
	ListDocuments(ctx context.Context) ([]DocumentMeta, error)

	// DeleteDocument remove um documento pelo ID.
	// RN-21: Ao deletar, o caller é responsável por remover os chunks do Qdrant também.
	DeleteDocument(ctx context.Context, id string) error

	// GetDocumentByName busca um documento pelo nome sanitizado.
	// Usado na detecção de duplicatas (RN-04).
	GetDocumentByName(ctx context.Context, name string) (*DocumentMeta, error)

	// Close fecha a conexão com o banco de dados.
	Close() error
}
