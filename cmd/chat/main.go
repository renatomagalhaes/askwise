// AskWise Chat — CLI interativo para perguntas sobre a base de conhecimento.
//
// Este é o ponto de entrada do chat no terminal. O usuário digita perguntas
// e recebe respostas geradas pelo pipeline RAG, baseadas nos documentos
// enviados via API.
//
// Spec dirigindo: RF-06, RN-17 a RN-20, design/01-ARQUITETURA.md §2.2
// ADR-001: Roda dentro de container Docker, sem Go local.
//
// Uso:
//
//	make chat     # via Docker Compose
//
// Implementação completa na Etapa 6 do PLAN.md.
// Nesta etapa (Etapa 0), apenas exibe uma mensagem de placeholder.
package main

import (
	"fmt"
	"log/slog"

	"github.com/renatomagalhaes/askwise/internal/logger"
)

func main() {
	logger.SetDefault("chat")

	slog.Info("AskWise Chat starting")

	fmt.Println()
	fmt.Println("  ╔══════════════════════════════════════════╗")
	fmt.Println("  ║          AskWise Chat v0.1.0             ║")
	fmt.Println("  ╠══════════════════════════════════════════╣")
	fmt.Println("  ║                                          ║")
	fmt.Println("  ║  Chat será implementado na Etapa 6.      ║")
	fmt.Println("  ║  Por enquanto, use a API para upload:     ║")
	fmt.Println("  ║                                          ║")
	fmt.Println("  ║  make upload FILE=documento.pdf           ║")
	fmt.Println("  ║                                          ║")
	fmt.Println("  ╚══════════════════════════════════════════╝")
	fmt.Println()

	slog.Info("AskWise Chat exiting (not yet implemented)")
}
