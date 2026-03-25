// AskWise Server — API HTTP para gerenciamento de documentos.
//
// Este é o ponto de entrada da API REST que permite upload, listagem e
// remoção de documentos da base de conhecimento.
//
// Spec dirigindo: RF-01, RF-08, RF-09, design/03-API-DESIGN.md
// ADR-001: Roda dentro de container Docker, sem Go local.
//
// Uso:
//
//	make up       # via Docker Compose
//	make health   # verifica se está rodando
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/renatomagalhaes/askwise/internal/chunker"
	"github.com/renatomagalhaes/askwise/internal/config"
	"github.com/renatomagalhaes/askwise/internal/document"
	"github.com/renatomagalhaes/askwise/internal/embedding"
	"github.com/renatomagalhaes/askwise/internal/llm"
	"github.com/renatomagalhaes/askwise/internal/logger"
	"github.com/renatomagalhaes/askwise/internal/rag"
	"github.com/renatomagalhaes/askwise/internal/retriever"
	"github.com/renatomagalhaes/askwise/internal/storage"
	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

// version é a versão da API, exibida no health check.
const version = "0.1.0"

// embeddingDimension é o tamanho dos vetores do text-embedding-3-small.
const embeddingDimension = 1536

func main() {
	// ADR-002: Logger global JSON estruturado.
	logger.SetDefault("server")

	// Carrega configuração. LoadOrWarn não falha se OPENAI_API_KEY estiver ausente
	// — permite que o servidor suba para health check e debug.
	cfg := config.LoadOrWarn()

	slog.Info("configuration loaded",
		"port", cfg.ServerPort,
		"qdrant", cfg.QdrantAddr(),
		"sqlite_path", cfg.SQLitePath,
		"openai_configured", cfg.OpenAIConfigured(),
	)

	app := initApp(cfg)
	defer app.storage.Close()

	mux := http.NewServeMux()
	registerRoutes(mux, app)

	// Pipeline de middleware:
	//   Request → Recovery → Logger → Handler → Response
	handler := withRecovery(withRequestLogger(mux))

	addr := cfg.ServerAddr()
	slog.Info("server starting", "addr", addr, "version", version)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// initApp cria o App com todos os componentes inicializados.
func initApp(cfg *config.Config) *App {
	ctx := context.Background()

	// ADR-003: SQLite para metadados de documentos.
	store, err := storage.NewSQLite(cfg.SQLitePath)
	if err != nil {
		slog.Error("failed to initialize SQLite", "error", err, "path", cfg.SQLitePath)
		os.Exit(1)
	}
	slog.Info("sqlite initialized", "path", cfg.SQLitePath)

	// ADR-004: Qdrant como vector store.
	vs := vectorstore.NewQdrant(cfg.QdrantAddr())
	if err := vs.EnsureCollection(ctx, cfg.QdrantCollection, embeddingDimension); err != nil {
		slog.Warn("qdrant collection setup failed (will retry on first request)",
			"error", err,
			"collection", cfg.QdrantCollection,
		)
	}

	// ADR-005: OpenAI como provider de embeddings e LLM.
	embedder := embedding.NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)
	llmClient := llm.NewOpenAILLM(cfg.OpenAIAPIKey, cfg.OpenAIChatModel)

	registry := document.NewRegistry()
	chk := chunker.NewRecursiveChunker(cfg.RAGChunkSize, cfg.RAGChunkOverlap)

	ret := retriever.NewSemanticRetriever(
		embedder, vs, cfg.QdrantCollection, float32(cfg.RAGScoreThreshold),
	)

	ragOrchestrator := rag.NewRAG(
		registry, chk, embedder, vs, ret, llmClient, store,
		rag.Config{Collection: cfg.QdrantCollection},
	)

	slog.Info("all components initialized",
		"collection", cfg.QdrantCollection,
		"chunk_size", cfg.RAGChunkSize,
		"chunk_overlap", cfg.RAGChunkOverlap,
		"top_k", cfg.RAGTopK,
		"score_threshold", cfg.RAGScoreThreshold,
	)

	return &App{
		rag:      ragOrchestrator,
		storage:  store,
		registry: registry,
		cfg:      cfg,
	}
}

// registerRoutes configura todas as rotas da API.
// Go 1.22+: pattern "METHOD /path" restringe ao método HTTP.
// Spec: design/03-API-DESIGN.md
func registerRoutes(mux *http.ServeMux, app *App) {
	mux.HandleFunc("GET /api/v1/health", app.handleHealth())
	mux.HandleFunc("GET /api/v1/documents", app.handleListDocuments())
	mux.HandleFunc("GET /api/v1/documents/{id}", app.handleGetDocument())
	mux.HandleFunc("DELETE /api/v1/documents/{id}", app.handleDeleteDocument())

	// RN-02: Aplica limite de tamanho apenas na rota de upload.
	uploadHandler := withMaxFileSize(app.cfg.MaxFileSize, http.HandlerFunc(app.handleUpload()))
	mux.Handle("POST /api/v1/documents", uploadHandler)
}
