package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/renatomagalhaes/askwise/internal/config"
	"github.com/renatomagalhaes/askwise/internal/document"
	"github.com/renatomagalhaes/askwise/internal/rag"
	"github.com/renatomagalhaes/askwise/internal/storage"
)

// App agrupa as dependências compartilhadas por todos os handlers.
type App struct {
	rag      *rag.RAG
	storage  storage.Storage
	registry *document.Registry
	cfg      *config.Config
}

// --- Response types (design/03-API-DESIGN.md) ---

type healthResponse struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

type errorResponse struct {
	Error            string   `json:"error"`
	Message          string   `json:"message"`
	SupportedFormats []string `json:"supported_formats,omitempty"`
}

type uploadResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	FileType     string    `json:"file_type"`
	FileSize     int64     `json:"file_size"`
	ChunkCount   int       `json:"chunk_count"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	Message      string    `json:"message"`
}

type listResponse struct {
	Documents   []storage.DocumentMeta `json:"documents"`
	Total       int                    `json:"total"`
	TotalChunks int                    `json:"total_chunks"`
}

type deleteResponse struct {
	Message string `json:"message"`
}

// --- Handlers ---

// handleHealth verifica a saúde do servidor e dependências.
// Spec: design/03-API-DESIGN.md §2.1
func (app *App) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qdrantStatus := checkQdrant(app.cfg)
		sqliteStatus := "connected"
		openaiStatus := "not_configured"
		if app.cfg.OpenAIConfigured() {
			openaiStatus = "configured"
		}

		status := "healthy"
		httpCode := http.StatusOK
		if qdrantStatus != "connected" {
			status = "unhealthy"
			httpCode = http.StatusServiceUnavailable
		}

		writeJSON(w, httpCode, healthResponse{
			Status:  status,
			Version: version,
			Dependencies: map[string]string{
				"qdrant": qdrantStatus,
				"sqlite": sqliteStatus,
				"openai": openaiStatus,
			},
		})
	}
}

// handleUpload recebe um arquivo e inicia o pipeline de ingestão.
// Spec: design/03-API-DESIGN.md §2.2
// Validações: RN-01 (formato), RN-02 (tamanho), RN-03 (conteúdo vazio)
func (app *App) handleUpload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Lê o arquivo do multipart form.
		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "invalid_request",
				Message: "Campo 'file' é obrigatório. Envie via multipart/form-data.",
			})
			return
		}
		defer file.Close()

		filename := header.Filename
		fileSize := header.Size

		// RN-01: Valida formato do arquivo.
		if !app.registry.IsSupported(filename) {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:            "unsupported_format",
				Message:          fmt.Sprintf("Formato não suportado. Formatos aceitos: %s", strings.Join(app.registry.SupportedFormats(), ", ")),
				SupportedFormats: app.registry.SupportedFormats(),
			})
			return
		}

		// Executa o pipeline de ingestão.
		result, err := app.rag.Ingest(r.Context(), file, filename, fileSize)
		if err != nil {
			// RN-03: Arquivo sem conteúdo textual extraível.
			if strings.Contains(err.Error(), "não contém conteúdo textual") {
				writeJSON(w, http.StatusUnprocessableEntity, errorResponse{
					Error:   "empty_content",
					Message: "Não foi possível extrair texto do arquivo. Verifique se o arquivo não está vazio ou corrompido.",
				})
				return
			}

			slog.Error("ingest failed",
				"component", "server",
				"filename", filename,
				"error", err,
			)
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error:   "ingest_error",
				Message: fmt.Sprintf("Falha ao processar documento: %s", err.Error()),
			})
			return
		}

		// Busca metadados completos para a resposta.
		doc, err := app.storage.GetDocument(r.Context(), result.DocumentID)
		if err != nil {
			doc = &storage.DocumentMeta{
				ID:       result.DocumentID,
				Name:     result.FileName,
				FileType: result.FileType,
				FileSize: fileSize,
				Status:   "ready",
			}
		}

		writeJSON(w, http.StatusCreated, uploadResponse{
			ID:         doc.ID,
			Name:       doc.Name,
			FileType:   doc.FileType,
			FileSize:   doc.FileSize,
			ChunkCount: doc.ChunkCount,
			Status:     doc.Status,
			CreatedAt:  doc.CreatedAt,
			Message:    fmt.Sprintf("Documento processado com sucesso. %d chunks indexados.", doc.ChunkCount),
		})
	}
}

// handleListDocuments retorna todos os documentos indexados.
// Spec: design/03-API-DESIGN.md §2.3
func (app *App) handleListDocuments() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docs, err := app.storage.ListDocuments(r.Context())
		if err != nil {
			slog.Error("list documents failed", "component", "server", "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error:   "list_error",
				Message: "Falha ao listar documentos",
			})
			return
		}

		if docs == nil {
			docs = []storage.DocumentMeta{}
		}

		totalChunks := 0
		for _, d := range docs {
			totalChunks += d.ChunkCount
		}

		writeJSON(w, http.StatusOK, listResponse{
			Documents:   docs,
			Total:       len(docs),
			TotalChunks: totalChunks,
		})
	}
}

// handleGetDocument retorna os detalhes de um documento.
// Spec: design/03-API-DESIGN.md §2.4
func (app *App) handleGetDocument() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "invalid_request",
				Message: "ID do documento é obrigatório",
			})
			return
		}

		doc, err := app.storage.GetDocument(r.Context(), id)
		if err != nil || doc == nil {
			writeJSON(w, http.StatusNotFound, errorResponse{
				Error:   "not_found",
				Message: fmt.Sprintf("Documento com ID '%s' não encontrado", id),
			})
			return
		}

		writeJSON(w, http.StatusOK, doc)
	}
}

// handleDeleteDocument remove um documento e seus chunks.
// Spec: design/03-API-DESIGN.md §2.5
// RN-21: Integridade referencial — remove do VectorStore e do Storage.
func (app *App) handleDeleteDocument() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, errorResponse{
				Error:   "invalid_request",
				Message: "ID do documento é obrigatório",
			})
			return
		}

		result, err := app.rag.Delete(r.Context(), id)
		if err != nil {
			if strings.Contains(err.Error(), "não encontrado") {
				writeJSON(w, http.StatusNotFound, errorResponse{
					Error:   "not_found",
					Message: fmt.Sprintf("Documento com ID '%s' não encontrado", id),
				})
				return
			}

			slog.Error("delete failed", "component", "server", "doc_id", id, "error", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{
				Error:   "delete_error",
				Message: fmt.Sprintf("Falha ao remover documento: %s", err.Error()),
			})
			return
		}

		writeJSON(w, http.StatusOK, deleteResponse{
			Message: fmt.Sprintf("Documento '%s' removido com sucesso. %d chunks deletados.", result.FileName, result.ChunkCount),
		})
	}
}

// --- Helpers ---

// checkQdrant verifica se o Qdrant está acessível.
func checkQdrant(cfg *config.Config) string {
	url := fmt.Sprintf("http://%s/healthz", cfg.QdrantAddr())
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		slog.Debug("qdrant health check failed", "error", err)
		return "disconnected"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "connected"
	}
	return "disconnected"
}

// writeJSON serializa um valor como JSON e escreve na response.
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
