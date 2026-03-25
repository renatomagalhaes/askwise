package rag

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// DeleteResult é o resultado da remoção de um documento.
type DeleteResult struct {
	DocumentID string `json:"document_id"`
	FileName   string `json:"file_name"`
	ChunkCount int    `json:"chunk_count"`
}

// Delete remove um documento e todos os seus chunks/embeddings.
// RN-21: Integridade referencial — remove do VectorStore e do Storage.
//
// Spec: design/03-API-DESIGN.md §2.5 (Remover Documento)
func (r *RAG) Delete(ctx context.Context, docID string) (*DeleteResult, error) {
	start := time.Now()

	// Busca metadados para obter nome e chunk_count (para a resposta).
	doc, err := r.storage.GetDocument(ctx, docID)
	if err != nil {
		return nil, fmt.Errorf("documento %s não encontrado: %w", docID, err)
	}
	if doc == nil {
		return nil, fmt.Errorf("documento %s não encontrado", docID)
	}

	slog.Info("delete started",
		"component", "rag",
		"doc_id", docID,
		"filename", doc.Name,
	)

	// Remove chunks do VectorStore (Qdrant).
	if err := r.vectorStore.DeleteByDocID(ctx, r.collection, docID); err != nil {
		return nil, fmt.Errorf("falha ao remover chunks do vectorstore: %w", err)
	}

	// Remove metadados do Storage (SQLite).
	if err := r.storage.DeleteDocument(ctx, docID); err != nil {
		return nil, fmt.Errorf("falha ao remover metadados: %w", err)
	}

	slog.Info("delete completed",
		"component", "rag",
		"doc_id", docID,
		"filename", doc.Name,
		"chunks_deleted", doc.ChunkCount,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &DeleteResult{
		DocumentID: docID,
		FileName:   doc.Name,
		ChunkCount: doc.ChunkCount,
	}, nil
}
