package vectorstore

import "context"

// Point representa um vetor a ser armazenado no Qdrant.
// Spec: design/02-MODELO-DADOS.md §4 (Point)
type Point struct {
	// ID é o identificador único do ponto (UUID do chunk).
	ID string
	// Vector é o embedding de N dimensões (ex: 1536 para text-embedding-3-small).
	Vector []float32
	// Payload são os metadados associados ao vetor.
	// RN-08: Deve conter document_id, chunk_index, text, file_name, file_type.
	Payload map[string]any
}

// SearchResult representa um resultado de busca vetorial.
// Spec: design/02-MODELO-DADOS.md §4 (SearchResult)
type SearchResult struct {
	// ID é o UUID do ponto encontrado.
	ID string
	// Score é a similaridade (cosine) com o vetor de consulta (0.0 a 1.0).
	Score float32
	// Payload são os metadados associados ao ponto.
	Payload map[string]any
}

// VectorStore define a interface para armazenamento e busca de vetores.
// Spec: design/01-ARQUITETURA.md §2.6
//
// A interface permite trocar Qdrant por outro vector store (ex: Pgvector)
// sem alterar o restante do código.
type VectorStore interface {
	// EnsureCollection garante que a collection existe com a configuração correta.
	// Se já existir, não faz nada (idempotente).
	EnsureCollection(ctx context.Context, name string, dimension int) error

	// Upsert insere ou atualiza pontos na collection.
	// Os pontos existentes com mesmo ID são substituídos.
	Upsert(ctx context.Context, collection string, points []Point) error

	// Search busca os topK pontos mais similares ao vetor fornecido.
	// Retorna resultados ordenados por score decrescente.
	Search(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error)

	// DeleteByDocID remove todos os pontos cujo payload contém o document_id especificado.
	// RN-21: Usado ao deletar um documento para remover seus chunks.
	DeleteByDocID(ctx context.Context, collection string, docID string) error
}
