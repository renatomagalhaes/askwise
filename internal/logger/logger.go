package logger

import (
	"io"
	"log/slog"
	"os"
)

// New cria um logger JSON estruturado para o componente especificado.
//
// O campo "component" é automaticamente adicionado a todos os logs,
// permitindo filtrar por origem (ex: "server", "chat", "rag", "storage").
//
// ADR-002: Usa slog.NewJSONHandler direcionado para os.Stdout por padrão.
func New(component string) *slog.Logger {
	return NewCustom(component, os.Stdout, slog.LevelDebug)
}

// NewCustom permite criar um logger com saída e nível específicos.
func NewCustom(component string, out io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler).With("component", component)
}

// SetDefault configura o logger global do slog para JSON.
// Deve ser chamado uma vez no início do programa (main).
func SetDefault(component string) {
	slog.SetDefault(New(component))
}

// SetDefaultCustom configura o logger global com saída e nível específicos.
func SetDefaultCustom(component string, out io.Writer, level slog.Level) {
	slog.SetDefault(NewCustom(component, out, level))
}

