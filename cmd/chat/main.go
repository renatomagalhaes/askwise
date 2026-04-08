// AskWise Chat — CLI interativo para perguntas sobre a base de conhecimento.
//
// O usuário digita perguntas e recebe respostas geradas pelo pipeline RAG,
// baseadas nos documentos enviados via API. As fontes são citadas ao final.
//
// Spec dirigindo: RF-06, RN-17 a RN-20, design/01-ARQUITETURA.md §2.2
// ADR-001: Roda dentro de container Docker, sem Go local.
//
// Uso:
//
//	make chat     # via Docker Compose
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/renatomagalhaes/askwise/internal/chunker"
	"github.com/renatomagalhaes/askwise/internal/config"
	"github.com/renatomagalhaes/askwise/internal/document"
	"github.com/renatomagalhaes/askwise/internal/embedding"
	llmpkg "github.com/renatomagalhaes/askwise/internal/llm"
	"github.com/renatomagalhaes/askwise/internal/logger"
	"github.com/renatomagalhaes/askwise/internal/rag"
	"github.com/renatomagalhaes/askwise/internal/retriever"
	"github.com/renatomagalhaes/askwise/internal/storage"
	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

const version = "0.1.0"

// maxHistoryMessages é o número máximo de mensagens mantidas no histórico.
// RN-18: Manter as últimas 10 mensagens como contexto.
const maxHistoryMessages = 10

// embeddingDimension é o tamanho dos vetores do text-embedding-3-small.
const embeddingDimension = 1536

// Cores ANSI para o terminal.
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorGray   = "\033[90m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorMagenta = "\033[35m"
)

func main() {
	// No chat CLI, silenciamos os logs no STDOUT/STDERR para não poluir a interface.
	// RF-06, RN-20: O feedback para o usuário é feito via print/spinner, não logs.
	logger.SetDefaultCustom("chat", io.Discard, slog.LevelInfo)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n  ❌ Erro de configuração: %s\n", err)
		fmt.Fprintf(os.Stderr, "     Configure o arquivo .env e tente novamente.\n\n")
		os.Exit(1)
	}

	ragOrch, store := initComponents(cfg)
	defer store.Close()

	printBanner()
	runLoop(ragOrch, store, cfg)
}

// initComponents inicializa todos os componentes do pipeline RAG.
func initComponents(cfg *config.Config) (*rag.RAG, storage.Storage) {
	ctx := context.Background()

	store, err := storage.NewSQLite(cfg.SQLitePath)
	if err != nil {
		slog.Error("failed to initialize SQLite", "error", err)
		fmt.Fprintf(os.Stderr, "\n  ❌ Falha ao inicializar SQLite: %s\n\n", err)
		os.Exit(1)
	}

	vs := vectorstore.NewQdrant(cfg.QdrantAddr())
	if err := vs.EnsureCollection(ctx, cfg.QdrantCollection, embeddingDimension); err != nil {
		slog.Warn("qdrant collection setup failed", "error", err)
	}

	embedder := embedding.NewOpenAIEmbedder(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)
	llmClient := llmpkg.NewOpenAILLM(cfg.OpenAIAPIKey, cfg.OpenAIChatModel)
	registry := document.NewRegistry()
	chk := chunker.NewRecursiveChunker(cfg.RAGChunkSize, cfg.RAGChunkOverlap)
	ret := retriever.NewSemanticRetriever(embedder, vs, cfg.QdrantCollection, float32(cfg.RAGScoreThreshold))

	ragOrch := rag.NewRAG(
		registry, chk, embedder, vs, ret, llmClient, store,
		rag.Config{Collection: cfg.QdrantCollection},
	)

	slog.Info("chat components initialized",
		"collection", cfg.QdrantCollection,
		"model", cfg.OpenAIChatModel,
	)

	return ragOrch, store
}

// printBanner exibe o cabeçalho do chat de forma alinhada e colorida.
func printBanner() {
	bannerWidth := 46
	fmt.Println()
	// Topo
	fmt.Printf("  %s╔%s╗%s\n", colorYellow, strings.Repeat("═", bannerWidth), colorReset)

	// Título: AskWise Chat vX.X.X
	title := fmt.Sprintf("AskWise Chat v%s", version)
	// Vamos centralizar de forma simples ou manter fixo.
	// ║          AskWise Chat v0.1.0                 ║
	fmt.Printf("  %s║%s%s%s%s%s%s║%s\n",
		colorYellow,
		strings.Repeat(" ", 10), // 10 espaços iniciais
		colorBold, title, colorReset,
		strings.Repeat(" ", bannerWidth-10-len(title)), // Resto do preenchimento
		colorYellow, colorReset)

	// Divisor
	fmt.Printf("  %s╠%s╣%s\n", colorYellow, strings.Repeat("═", bannerWidth), colorReset)

	// Linha 1: Pergunte sobre...
	line1 := "Pergunte sobre os documentos da sua base."
	fmt.Printf("  %s║%s  %-*s  %s║%s\n",
		colorYellow, colorReset, bannerWidth-4, line1, colorYellow, colorReset)

	// Linha 2: Digite /help...
	line2Part1 := "Digite "
	line2Part2 := "/help"
	line2Part3 := " para ver os comandos."
	// Calculamos o espaço puro: "Digite " (7) + "/help" (5) + " para ver os comandos." (22) + 2 spaces = 36
	// Total visible: len(line2Part1) + len(line2Part2) + len(line2Part3) + 4 (spaces) = 7 + 5 + 23 + 4 = 39
	// Precisamos de bannerWidth = 46.
	line2VisibleLen := len(line2Part1) + len(line2Part2) + len(line2Part3) + 4
	line2Padding := strings.Repeat(" ", bannerWidth-line2VisibleLen)

	fmt.Printf("  %s║%s  %s%s%s%s%s  %s%s║%s\n",
		colorYellow, colorReset,
		line2Part1, colorCyan, line2Part2, colorReset, line2Part3,
		line2Padding, colorYellow, colorReset)

	// Base
	fmt.Printf("  %s╚%s╝%s\n", colorYellow, strings.Repeat("═", bannerWidth), colorReset)
	fmt.Println()
}

// runLoop é o loop principal do chat interativo.
// RN-19: Suporta comandos especiais iniciados com /.
func runLoop(ragOrch *rag.RAG, store storage.Storage, cfg *config.Config) {
	scanner := bufio.NewScanner(os.Stdin)
	// Aumenta buffer para permitir perguntas longas.
	scanner.Buffer(make([]byte, 0, 64*1024), 64*1024)

	var history []llmpkg.Message
	ctx := context.Background()

	for {
		fmt.Printf("  %s%svocê>%s ", colorBlue, colorBold, colorReset)

		if !scanner.Scan() {
			// EOF (Ctrl+D) ou erro de leitura.
			fmt.Println()
			printGoodbye()
			return
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// RN-19: Comandos especiais.
		if strings.HasPrefix(input, "/") {
			shouldExit := handleCommand(input, &history, store, cfg, ctx)
			if shouldExit {
				return
			}
			continue
		}

		// Executa o pipeline de consulta RAG.
		response := queryRAG(ctx, ragOrch, input, history)
		if response == nil {
			continue
		}

		// Exibe a resposta formatada.
		printResponse(response)

		// RN-18: Atualiza histórico da conversa.
		history = appendToHistory(history, input, response.Answer)
	}
}

// handleCommand processa comandos especiais iniciados com /.
// RN-19: /help, /clear, /quit, /exit, /sources, /stats.
// Retorna true se o chat deve encerrar.
func handleCommand(input string, history *[]llmpkg.Message, store storage.Storage, cfg *config.Config, ctx context.Context) bool {
	cmd := strings.ToLower(strings.Fields(input)[0])

	switch cmd {
	case "/quit", "/exit":
		printGoodbye()
		return true

	case "/help":
		printHelp()

	case "/clear":
		*history = nil
		fmt.Println()
		fmt.Println("  ✓ Histórico limpo.")
		fmt.Println()

	case "/sources":
		printSources(ctx, store)

	case "/stats":
		printStats(ctx, store, cfg)

	default:
		fmt.Println()
		fmt.Printf("  ⚠ Comando desconhecido: %s\n", cmd)
		fmt.Println("  Digite /help para ver os comandos disponíveis.")
		fmt.Println()
	}

	return false
}

// queryRAG executa a consulta ao pipeline RAG com feedback visual.
// RN-20: Mostra indicadores de progresso.
func queryRAG(ctx context.Context, ragOrch *rag.RAG, question string, history []llmpkg.Message) *rag.QueryResponse {
	fmt.Println()

	// RN-20: Feedback visual durante processamento.
	done := make(chan struct{})
	go showSpinner("  Pensando", done)

	start := time.Now()
	response, err := ragOrch.Query(ctx, question, history)
	close(done)

	// Limpa a linha do spinner.
	fmt.Print("\r\033[K")

	if err != nil {
		slog.Error("query failed", "error", err)
		fmt.Printf("  ❌ Erro ao processar pergunta: %s\n\n", err)
		return nil
	}

	slog.Debug("query timing", "duration_ms", time.Since(start).Milliseconds())
	return response
}

// printResponse exibe a resposta e as fontes de forma formatada e colorida.
// RN-14: Citação de fontes ao final da resposta.
func printResponse(resp *rag.QueryResponse) {
	fmt.Printf("  %s──────────────────────────────────────────────%s\n", colorGray, colorReset)
	fmt.Println()

	// Indenta cada linha da resposta para alinhamento visual e usa cor Ciano.
	lines := strings.Split(resp.Answer, "\n")
	for _, line := range lines {
		fmt.Printf("  %s%s%s\n", colorCyan, line, colorReset)
	}

	// RN-14: Lista fontes em cor escura/cinza.
	if len(resp.Sources) > 0 {
		fmt.Println()
		fmt.Printf("  %s📎 Fontes:%s\n", colorBold, colorReset)
		for _, s := range resp.Sources {
			fmt.Printf("     %s• %-30s (relevância: %.0f%%)%s\n", colorGray, s.FileName, s.Score*100, colorReset)
		}
	}

	fmt.Println()
	fmt.Printf("  %s──────────────────────────────────────────────%s\n", colorGray, colorReset)
	fmt.Println()
}

// appendToHistory adiciona a pergunta e resposta ao histórico.
// RN-18: Mantém no máximo maxHistoryMessages mensagens.
func appendToHistory(history []llmpkg.Message, question, answer string) []llmpkg.Message {
	history = append(history,
		llmpkg.Message{Role: llmpkg.RoleUser, Content: question},
		llmpkg.Message{Role: llmpkg.RoleAssistant, Content: answer},
	)

	// Trunca mantendo as mensagens mais recentes.
	if len(history) > maxHistoryMessages {
		history = history[len(history)-maxHistoryMessages:]
	}

	return history
}

// --- Comandos especiais ---

func printHelp() {
	fmt.Println()
	fmt.Printf("  %s📖 Comandos disponíveis:%s\n", colorBold, colorReset)
	fmt.Println()
	fmt.Printf("     %s/help%s       Mostra esta mensagem\n", colorCyan, colorReset)
	fmt.Printf("     %s/sources%s    Lista documentos indexados\n", colorCyan, colorReset)
	fmt.Printf("     %s/stats%s      Mostra estatísticas da base\n", colorCyan, colorReset)
	fmt.Printf("     %s/clear%s      Limpa o histórico da conversa\n", colorCyan, colorReset)
	fmt.Printf("     %s/quit%s       Sai do chat\n", colorCyan, colorReset)
	fmt.Printf("     %s/exit%s       Sai do chat\n", colorCyan, colorReset)
	fmt.Println()
}

// printSources lista os documentos indexados no sistema.
func printSources(ctx context.Context, store storage.Storage) {
	docs, err := store.ListDocuments(ctx)
	if err != nil {
		fmt.Printf("\n  ❌ Erro ao listar documentos: %s\n\n", err)
		return
	}

	fmt.Println()
	if len(docs) == 0 {
		fmt.Printf("  %s📂 Nenhum documento indexado.%s\n", colorYellow, colorReset)
		fmt.Printf("     Use '%smake upload FILE=doc.pdf%s' para enviar documentos.\n", colorCyan, colorReset)
	} else {
		fmt.Printf("  %s📂 Documentos indexados (%d):%s\n", colorBold, len(docs), colorReset)
		fmt.Println()
		for _, d := range docs {
			fmt.Printf("     • %s%-30s%s  %s[%s]%s  %s%d chunks%s\n", colorBold, d.Name, colorReset, colorCyan, d.FileType, colorReset, colorGray, d.ChunkCount, colorReset)
		}
	}
	fmt.Println()
}

// printStats exibe estatísticas da base de conhecimento.
func printStats(ctx context.Context, store storage.Storage, cfg *config.Config) {
	docs, err := store.ListDocuments(ctx)
	if err != nil {
		fmt.Printf("\n  ❌ Erro ao obter estatísticas: %s\n\n", err)
		return
	}

	totalChunks := 0
	for _, d := range docs {
		totalChunks += d.ChunkCount
	}

	fmt.Println()
	fmt.Println("  📊 Estatísticas:")
	fmt.Println()
	fmt.Printf("     Documentos:    %d\n", len(docs))
	fmt.Printf("     Chunks:        %d\n", totalChunks)
	fmt.Printf("     Modelo LLM:    %s\n", cfg.OpenAIChatModel)
	fmt.Printf("     Modelo Embed:  %s\n", cfg.OpenAIEmbeddingModel)
	fmt.Printf("     Collection:    %s\n", cfg.QdrantCollection)
	fmt.Printf("     Chunk size:    %d\n", cfg.RAGChunkSize)
	fmt.Printf("     Chunk overlap: %d\n", cfg.RAGChunkOverlap)
	fmt.Printf("     Top-K:         %d\n", cfg.RAGTopK)
	fmt.Printf("     Score min:     %.2f\n", cfg.RAGScoreThreshold)
	fmt.Println()
}

func printGoodbye() {
	fmt.Println()
	fmt.Println("  👋 Até mais! Obrigado por usar o AskWise.")
	fmt.Println()
}

// --- Spinner ---

// showSpinner exibe uma animação de progresso até o canal ser fechado.
// RN-20: Feedback visual durante processamento.
func showSpinner(prefix string, done <-chan struct{}) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	for {
		select {
		case <-done:
			return
		default:
			fmt.Printf("\r  %s %s", frames[i%len(frames)], prefix)
			i++
			time.Sleep(80 * time.Millisecond)
		}
	}
}
