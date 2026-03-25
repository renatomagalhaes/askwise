package rag

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/renatomagalhaes/askwise/internal/llm"
	"github.com/renatomagalhaes/askwise/internal/retriever"
)

// systemPrompt contém as instruções de comportamento para a LLM.
// Implementa as regras de negócio:
// RN-13: Resposta baseada em contexto (não inventar informações).
// RN-14: Citação de fontes ao final da resposta.
// RN-15: Responder no mesmo idioma da pergunta.
// RN-16: Tom profissional, direto e acionável.
const systemPrompt = `Você é o AskWise, um assistente de suporte técnico sênior.

REGRAS OBRIGATÓRIAS:
1. Responda APENAS com base no contexto fornecido abaixo. NÃO invente informações.
2. Se a informação não estiver no contexto, diga: "Não encontrei informações sobre isso na base de conhecimento."
3. Responda no mesmo idioma da pergunta do usuário.
4. Seja profissional, direto e acionável — dê respostas claras e passo-a-passo quando possível.
5. Ao final da resposta, NÃO liste as fontes — o sistema faz isso automaticamente.

CONTEXTO (documentos da base de conhecimento):
---
%s
---`

// Query executa o pipeline de consulta RAG:
// Retrieve → Build prompt → ChatCompletion → Response.
//
// Spec: design/01-ARQUITETURA.md §3.2 (Pipeline de Consulta)
func (r *RAG) Query(ctx context.Context, question string, history []llm.Message) (*QueryResponse, error) {
	start := time.Now()

	slog.Info("query started",
		"component", "rag",
		"question_length", len(question),
		"history_messages", len(history),
	)

	// 1. Retrieve: busca chunks relevantes.
	// RN-10: Top-K=5 é configurado no Retriever.
	chunks, err := r.retriever.Retrieve(ctx, question, 5)
	if err != nil {
		return nil, fmt.Errorf("falha na recuperação de contexto: %w", err)
	}

	// RN-17: Se nenhum chunk relevante for encontrado, retorna mensagem padrão.
	if len(chunks) == 0 {
		slog.Info("no relevant chunks found",
			"component", "rag",
			"question", question,
		)
		return &QueryResponse{
			Answer:  "Não encontrei informações sobre isso na base de conhecimento. Tente reformular a pergunta ou verifique se o documento relevante já foi enviado.",
			Sources: nil,
		}, nil
	}

	// 2. Build prompt: monta as mensagens para a LLM.
	messages := buildMessages(question, chunks, history)

	// 3. ChatCompletion: gera a resposta.
	answer, err := r.llm.ChatCompletion(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("falha na geração de resposta: %w", err)
	}

	// Monta as fontes a partir dos chunks utilizados (RN-14).
	sources := chunksToSources(chunks)

	slog.Info("query completed",
		"component", "rag",
		"chunks_used", len(chunks),
		"sources", len(sources),
		"answer_length", len(answer),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &QueryResponse{
		Answer:  answer,
		Sources: sources,
	}, nil
}

// buildMessages monta a lista de mensagens para a LLM.
// Inclui: system prompt com contexto, histórico e pergunta atual.
func buildMessages(question string, chunks []retriever.RetrievedChunk, history []llm.Message) []llm.Message {
	contextText := formatContext(chunks)

	sysMsg := llm.Message{
		Role:    llm.RoleSystem,
		Content: fmt.Sprintf(systemPrompt, contextText),
	}

	messages := make([]llm.Message, 0, 1+len(history)+1)
	messages = append(messages, sysMsg)

	// RN-18: Inclui histórico da conversa (últimas N mensagens).
	messages = append(messages, history...)

	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: question,
	})

	return messages
}

// formatContext formata os chunks em texto para inclusão no prompt.
func formatContext(chunks []retriever.RetrievedChunk) string {
	var sb strings.Builder
	for i, c := range chunks {
		sb.WriteString(fmt.Sprintf("[Fonte: %s, trecho %d]\n", c.FileName, c.Index))
		sb.WriteString(c.Text)
		if i < len(chunks)-1 {
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

// chunksToSources converte chunks em fontes, deduplicando por arquivo.
func chunksToSources(chunks []retriever.RetrievedChunk) []Source {
	var sources []Source
	seen := make(map[string]int)

	for _, c := range chunks {
		if idx, exists := seen[c.FileName]; exists {
			if c.Score > sources[idx].Score {
				sources[idx].Score = c.Score
			}
		} else {
			seen[c.FileName] = len(sources)
			sources = append(sources, Source{
				FileName: c.FileName,
				FileType: c.FileType,
				Index:    c.Index,
				Score:    c.Score,
			})
		}
	}

	return sources
}
