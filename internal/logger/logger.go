package logger

import (
	"log/slog"
	"os"
)

// New cria um logger JSON estruturado para o componente especificado.
//
// O campo "component" é automaticamente adicionado a todos os logs,
// permitindo filtrar por origem (ex: "server", "chat", "rag", "storage").
//
// ADR-002: Usa slog.JSONHandler direcionado para os.Stdout.
// Os logs de nível WARN e ERROR devem ser tratados pelo caller usando
// as funções log.Warn() e log.Error() que, combinadas com o handler,
// permitem redirecionamento via infraestrutura (Docker, systemd, etc.).
//
// Em containers Docker, tanto STDOUT quanto STDERR são capturados por
// `docker logs`, então usamos STDOUT como destino único para simplificar
// e evitar interleaving de streams.
func New(component string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		// Nível mínimo de log. Em produção poderia ser INFO,
		// mas para uma PoC educativa mantemos DEBUG para ver tudo.
		Level: slog.LevelDebug,
	})

	// slog.With() retorna um logger com campos pré-definidos.
	// "component" aparece em TODOS os logs deste logger, facilitando
	// filtrar por origem: jq 'select(.component == "server")'
	return slog.New(handler).With("component", component)
}

// NewDefault configura o logger global do slog para JSON.
// Deve ser chamado uma vez no início do programa (main).
//
// Após chamar SetDefault, qualquer uso de slog.Info(), slog.Error(), etc.
// em qualquer parte do código usará o formato JSON automaticamente.
func SetDefault(component string) {
	logger := New(component)
	slog.SetDefault(logger)
}
