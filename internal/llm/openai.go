package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	// defaultChatURL é o endpoint da API de chat completion da OpenAI.
	defaultChatURL = "https://api.openai.com/v1/chat/completions"
)

// OpenAILLM implementa LLM usando a API de chat completion da OpenAI.
// ADR-005: OpenAI como provider de LLM.
type OpenAILLM struct {
	apiKey string
	model  string
	url    string
	client *http.Client
}

// NewOpenAILLM cria um novo OpenAILLM.
// apiKey é a chave da API da OpenAI (OPENAI_API_KEY).
// model é o modelo de chat (ex: "gpt-4o-mini").
func NewOpenAILLM(apiKey, model string) *OpenAILLM {
	return &OpenAILLM{
		apiKey: apiKey,
		model:  model,
		url:    defaultChatURL,
		client: &http.Client{Timeout: 60 * time.Second},
	}
}

// WithURL substitui a URL da API (usado em testes com mock server).
func (l *OpenAILLM) WithURL(url string) *OpenAILLM {
	l.url = url
	return l
}

// WithHTTPClient substitui o http.Client (usado em testes).
func (l *OpenAILLM) WithHTTPClient(client *http.Client) *OpenAILLM {
	l.client = client
	return l
}

// ChatCompletion envia mensagens à API e retorna a resposta gerada pelo modelo.
// Timeout de 60s para acomodar respostas longas da LLM.
func (l *OpenAILLM) ChatCompletion(ctx context.Context, messages []Message) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("lista de mensagens vazia")
	}

	start := time.Now()

	reqBody := chatRequest{
		Model:    l.model,
		Messages: messages,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("falha ao serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("falha ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := l.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("falha na chamada à API de chat: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("falha ao ler resposta da API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp chatErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("API de chat retornou %d: %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("API de chat retornou %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("falha ao parsear resposta da API: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API retornou resposta sem choices")
	}

	content := chatResp.Choices[0].Message.Content

	slog.Info("chat completion generated",
		"component", "llm",
		"model", chatResp.Model,
		"prompt_tokens", chatResp.Usage.PromptTokens,
		"completion_tokens", chatResp.Usage.CompletionTokens,
		"total_tokens", chatResp.Usage.TotalTokens,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return content, nil
}

// chatRequest é o corpo da requisição para a API de chat completion.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// chatResponse é a resposta da API de chat completion.
type chatResponse struct {
	Choices []chatChoice `json:"choices"`
	Model   string       `json:"model"`
	Usage   chatUsage    `json:"usage"`
}

type chatChoice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type chatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// chatErrorResponse captura erros da API da OpenAI.
type chatErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
