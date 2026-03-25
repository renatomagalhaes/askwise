package rag

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/renatomagalhaes/askwise/internal/storage"
	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

// IngestResult é o resultado do pipeline de ingestão.
type IngestResult struct {
	DocumentID string `json:"document_id"`
	FileName   string `json:"file_name"`
	FileType   string `json:"file_type"`
	FileSize   int64  `json:"file_size"`
	ChunkCount int    `json:"chunk_count"`
}

// Ingest executa o pipeline de ingestão completo:
// Parse → Chunk → Embed → Upsert + SaveDocument.
//
// Spec: design/01-ARQUITETURA.md §3.1 (Pipeline de Ingestão)
func (r *RAG) Ingest(ctx context.Context, reader io.Reader, filename string, fileSize int64) (*IngestResult, error) {
	start := time.Now()

	sanitizedName := sanitizeFilename(filename)
	ext := strings.TrimPrefix(filepath.Ext(sanitizedName), ".")
	ext = strings.ToLower(ext)

	slog.Info("ingest started",
		"component", "rag",
		"filename", sanitizedName,
		"original_name", filename,
		"file_size", fileSize,
	)

	// RN-04: Verifica duplicatas pelo nome. Se existir, remove o antigo.
	if err := r.handleDuplicate(ctx, sanitizedName); err != nil {
		return nil, fmt.Errorf("falha ao verificar duplicata: %w", err)
	}

	// Gera UUID para o novo documento.
	docID := uuid.New().String()

	// Salva metadados com status "processing".
	docMeta := &storage.DocumentMeta{
		ID:           docID,
		Name:         sanitizedName,
		OriginalName: filename,
		FileType:     ext,
		FileSize:     fileSize,
		Status:       "processing",
	}
	if err := r.storage.SaveDocument(ctx, docMeta); err != nil {
		return nil, fmt.Errorf("falha ao salvar metadados: %w", err)
	}

	// Executa o pipeline. Em caso de erro, atualiza status para "error".
	result, err := r.executeIngestPipeline(ctx, reader, sanitizedName, docID)
	if err != nil {
		r.markDocumentError(ctx, docMeta, err.Error())
		return nil, err
	}

	// Atualiza metadados com status "ready" e chunk_count.
	docMeta.ChunkCount = result.ChunkCount
	docMeta.Status = "ready"
	if err := r.storage.SaveDocument(ctx, docMeta); err != nil {
		return nil, fmt.Errorf("falha ao atualizar metadados: %w", err)
	}

	slog.Info("ingest completed",
		"component", "rag",
		"doc_id", docID,
		"filename", sanitizedName,
		"chunks", result.ChunkCount,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return result, nil
}

// executeIngestPipeline executa Parse → Chunk → Embed → Upsert.
func (r *RAG) executeIngestPipeline(ctx context.Context, reader io.Reader, filename, docID string) (*IngestResult, error) {
	// 1. Parse: extrai texto do arquivo.
	doc, err := r.parser.Parse(reader, filename)
	if err != nil {
		return nil, fmt.Errorf("falha no parsing de %s: %w", filename, err)
	}

	// 2. Chunk: divide texto em pedaços.
	chunks, err := r.chunker.Chunk(doc)
	if err != nil {
		return nil, fmt.Errorf("falha no chunking de %s: %w", filename, err)
	}

	// Preenche IDs dos chunks (design/02-MODELO-DADOS.md §6).
	texts := make([]string, len(chunks))
	for i := range chunks {
		chunks[i].DocumentID = docID
		chunks[i].ID = ChunkID(docID, chunks[i].Index)
		texts[i] = chunks[i].Text
	}

	// 3. Embed: gera vetores para todos os chunks.
	embeddings, err := r.embedder.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar embeddings para %s: %w", filename, err)
	}

	// 4. Upsert: salva vetores + metadados no VectorStore.
	points := make([]vectorstore.Point, len(chunks))
	for i, c := range chunks {
		points[i] = vectorstore.Point{
			ID:     c.ID,
			Vector: embeddings[i],
			Payload: map[string]any{
				"document_id": c.DocumentID,
				"chunk_index": c.Index,
				"text":        c.Text,
				"file_name":   c.FileName,
				"file_type":   c.FileType,
			},
		}
	}

	if err := r.vectorStore.Upsert(ctx, r.collection, points); err != nil {
		return nil, fmt.Errorf("falha ao armazenar vetores de %s: %w", filename, err)
	}

	return &IngestResult{
		DocumentID: docID,
		FileName:   filename,
		FileType:   doc.FileType,
		FileSize:   0, // preenchido pelo caller
		ChunkCount: len(chunks),
	}, nil
}

// handleDuplicate verifica se um documento com o mesmo nome já existe.
// RN-04: Se existir, remove o antigo (chunks + metadados) antes de reprocessar.
// RN-22: Idempotência — re-upload produz substituição completa.
func (r *RAG) handleDuplicate(ctx context.Context, name string) error {
	existing, err := r.storage.GetDocumentByName(ctx, name)
	if err != nil {
		return fmt.Errorf("falha ao verificar duplicata: %w", err)
	}
	if existing == nil {
		return nil
	}

	slog.Info("duplicate detected, removing old version",
		"component", "rag",
		"doc_id", existing.ID,
		"name", name,
	)

	// RN-21: Remove chunks do VectorStore.
	if err := r.vectorStore.DeleteByDocID(ctx, r.collection, existing.ID); err != nil {
		return fmt.Errorf("falha ao remover chunks antigos: %w", err)
	}

	// Remove metadados do SQLite.
	if err := r.storage.DeleteDocument(ctx, existing.ID); err != nil {
		return fmt.Errorf("falha ao remover documento antigo: %w", err)
	}

	return nil
}

// markDocumentError atualiza o status do documento para "error".
func (r *RAG) markDocumentError(ctx context.Context, doc *storage.DocumentMeta, errMsg string) {
	doc.Status = "error"
	doc.ErrorMessage = errMsg
	if err := r.storage.SaveDocument(ctx, doc); err != nil {
		slog.Error("failed to mark document as error",
			"component", "rag",
			"doc_id", doc.ID,
			"error", err,
		)
	}
}

// ChunkID gera um ID determinístico para um chunk baseado no documento e índice.
// Spec: design/02-MODELO-DADOS.md §6 (Estratégia de IDs)
//
// Usa UUID v5 (SHA1) para que re-uploads gerem os mesmos IDs,
// permitindo upsert sem duplicação no VectorStore.
func ChunkID(documentID string, chunkIndex int) string {
	namespace := uuid.MustParse(documentID)
	name := fmt.Sprintf("chunk-%d", chunkIndex)
	return uuid.NewSHA1(namespace, []byte(name)).String()
}

// sanitizeFilename limpa o nome do arquivo para armazenamento seguro.
// RN-05: Remover caracteres especiais, limitar tamanho.
var unsafeChars = regexp.MustCompile(`[^\w\-.]`)

func sanitizeFilename(name string) string {
	// Extrai apenas o nome do arquivo (sem path).
	name = filepath.Base(name)

	// Substitui espaços por hífens.
	name = strings.ReplaceAll(name, " ", "-")

	// Remove caracteres não seguros.
	name = unsafeChars.ReplaceAllString(name, "")

	// Remove hífens/pontos consecutivos.
	name = regexp.MustCompile(`-{2,}`).ReplaceAllString(name, "-")
	name = regexp.MustCompile(`\.{2,}`).ReplaceAllString(name, ".")

	// Limita tamanho.
	if len(name) > 255 {
		ext := filepath.Ext(name)
		name = name[:255-len(ext)] + ext
	}

	return strings.ToLower(name)
}
