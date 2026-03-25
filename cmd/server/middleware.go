package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// withRequestLogger registra cada request HTTP em JSON estruturado.
// ADR-002: Logs com campos contextuais (método, path, status, duração).
func withRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
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

// withRecovery captura panics e retorna HTTP 500 com mensagem genérica.
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", fmt.Sprintf("%v", err),
					"method", r.Method,
					"path", r.URL.Path,
				)
				writeJSON(w, http.StatusInternalServerError, errorResponse{
					Error:   "internal_error",
					Message: "Erro interno do servidor",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withMaxFileSize limita o tamanho do body da request.
// RN-02: Arquivos devem ter no máximo 10MB.
func withMaxFileSize(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > maxBytes {
			writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{
				Error:   "file_too_large",
				Message: fmt.Sprintf("Arquivo excede o limite de %dMB", maxBytes/(1024*1024)),
			})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

// responseWriter captura o status code para logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
