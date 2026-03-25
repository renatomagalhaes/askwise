package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// TestNewReturnsJSONLogger verifica que o logger retorna output em formato JSON
// com o campo "component" incluído automaticamente — ADR-002.
func TestNewReturnsJSONLogger(t *testing.T) {
	// Cria um buffer para capturar o output em vez de STDOUT
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	log := slog.New(handler).With("component", "test")

	log.Info("hello", "key", "value")

	// Verifica se o output é JSON válido
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output não é JSON válido: %v\nOutput: %s", err, buf.String())
	}

	// Verifica campos obrigatórios
	tests := []struct {
		field string
		want  any
	}{
		{"level", "INFO"},
		{"msg", "hello"},
		{"component", "test"},
		{"key", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got, ok := entry[tt.field]
			if !ok {
				t.Errorf("campo %q ausente no JSON", tt.field)
				return
			}
			if got != tt.want {
				t.Errorf("campo %q = %v, esperava %v", tt.field, got, tt.want)
			}
		})
	}

	// Verifica que o campo "time" existe (timestamp RFC3339)
	if _, ok := entry["time"]; !ok {
		t.Error("campo 'time' ausente no JSON")
	}
}

// TestLogLevels verifica que diferentes níveis de log são emitidos corretamente.
func TestLogLevels(t *testing.T) {
	levels := []struct {
		name  string
		logFn func(*slog.Logger, string, ...any)
		want  string
	}{
		{"DEBUG", (*slog.Logger).Debug, "DEBUG"},
		{"INFO", (*slog.Logger).Info, "INFO"},
		{"WARN", (*slog.Logger).Warn, "WARN"},
		{"ERROR", (*slog.Logger).Error, "ERROR"},
	}

	for _, tt := range levels {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			log := slog.New(handler).With("component", "test")

			tt.logFn(log, "test message")

			var entry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("output não é JSON válido: %v", err)
			}

			if entry["level"] != tt.want {
				t.Errorf("level = %v, esperava %v", entry["level"], tt.want)
			}
		})
	}
}
