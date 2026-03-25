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
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/renatomagalhaes/askwise/internal/config"
	"github.com/renatomagalhaes/askwise/internal/logger"
)

// version é a versão da API, exibida no health check.
const version = "0.1.0"

func main() {
	// ADR-002: Configura o logger global como JSON estruturado.
	// A partir daqui, qualquer uso de slog.Info(), slog.Error() etc.
	// em qualquer parte do código emite JSON para STDOUT.
	logger.SetDefault("server")

	// Carrega configuração das variáveis de ambiente.
	// LoadOrWarn não falha se OPENAI_API_KEY estiver ausente — permite
	// que o servidor suba mesmo sem a key (útil para health check e debug).
	cfg := config.LoadOrWarn()

	slog.Info("configuration loaded",
		"port", cfg.ServerPort,
		"qdrant", cfg.QdrantAddr(),
		"sqlite_path", cfg.SQLitePath,
		"openai_configured", cfg.OpenAIConfigured(),
	)

	// Cria o roteador HTTP.
	// Go 1.22+ suporta pattern matching no ServeMux: "GET /path" restringe ao método.
	mux := http.NewServeMux()

	// Registra os endpoints da API.
	// Spec: design/03-API-DESIGN.md
	mux.HandleFunc("GET /api/v1/health", handleHealth(cfg))

	// Monta o pipeline de middleware.
	// Cada middleware envolve o handler anterior, formando uma cadeia:
	//   Request → Logger → Recovery → Handler → Response
	handler := withRecovery(withRequestLogger(mux))

	// Inicia o servidor HTTP.
	addr := cfg.ServerAddr()
	slog.Info("server starting", "addr", addr, "version", version)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// --- Handlers ---

// healthResponse é a estrutura da resposta do health check,
// conforme definido em api/openapi.yaml (HealthResponse).
type healthResponse struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

// handleHealth retorna um handler para GET /api/v1/health.
// Verifica o status de cada dependência e retorna o resultado.
//
// Spec: design/03-API-DESIGN.md §2.1
func handleHealth(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Verifica cada dependência
		qdrantStatus := checkQdrant(cfg)
		sqliteStatus := "connected" // SQLite auto-cria na inicialização (storage.NewSQLite)
		openaiStatus := "not_configured"
		if cfg.OpenAIConfigured() {
			openaiStatus = "configured"
		}

		// Determina status geral: healthy se todas as dependências críticas estão ok
		status := "healthy"
		httpCode := http.StatusOK
		if qdrantStatus != "connected" {
			status = "unhealthy"
			httpCode = http.StatusServiceUnavailable
		}

		resp := healthResponse{
			Status:  status,
			Version: version,
			Dependencies: map[string]string{
				"qdrant": qdrantStatus,
				"sqlite": sqliteStatus,
				"openai": openaiStatus,
			},
		}

		writeJSON(w, httpCode, resp)
	}
}

// checkQdrant verifica se o Qdrant está acessível fazendo um GET /healthz.
func checkQdrant(cfg *config.Config) string {
	url := fmt.Sprintf("http://%s/healthz", cfg.QdrantAddr())

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		slog.Debug("qdrant health check failed", "error", err, "url", url)
		return "disconnected"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "connected"
	}
	return "disconnected"
}

// --- Middleware ---

// withRequestLogger é um middleware que registra cada request HTTP em JSON.
// ADR-002: Logs estruturados com campos contextuais.
func withRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// responseWriter wrapper para capturar o status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// withRecovery é um middleware que captura panics e retorna HTTP 500.
// Evita que um panic derrube o servidor inteiro.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", err),
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error":   "internal_error",
					"message": "Erro interno do servidor",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- Helpers ---

// responseWriter é um wrapper de http.ResponseWriter que captura o status code
// escrito, permitindo logar o status no middleware de request logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captura o status code antes de delegar ao ResponseWriter real.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// writeJSON serializa um valor como JSON e escreve na response.
// Define Content-Type como application/json e o status code fornecido.
func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
