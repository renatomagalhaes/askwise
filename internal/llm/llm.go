package llm

import "context"

// Role identifica o papel de quem enviou uma mensagem no chat.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message representa uma mensagem no formato de chat completion.
// Spec: design/01-ARQUITETURA.md §2.8
type Message struct {
	// Role é o papel: "system", "user" ou "assistant".
	Role string `json:"role"`

	// Content é o conteúdo textual da mensagem.
	Content string `json:"content"`
}

// LLM define a interface para modelos de linguagem (chat completion).
// Spec: design/01-ARQUITETURA.md §2.8
//
// A interface permite trocar o provider sem impacto no pipeline RAG.
// As regras de negócio (RN-13 a RN-16) são aplicadas via system prompt
// pelo orquestrador, não pelo client LLM diretamente.
type LLM interface {
	// ChatCompletion envia uma lista de mensagens e retorna a resposta do modelo.
	// A primeira mensagem geralmente é o system prompt com instruções de comportamento.
	ChatCompletion(ctx context.Context, messages []Message) (string, error)
}
