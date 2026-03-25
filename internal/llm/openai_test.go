package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockChatServer cria um servidor HTTP mock que simula a API de chat completion.
// Responde com o texto fornecido em responseText.
func mockChatServer(t *testing.T, responseText string, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"message": "missing API key"},
			})
			return
		}

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if statusCode != http.StatusOK {
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"message": "simulated error",
					"type":    "server_error",
				},
			})
			return
		}

		resp := chatResponse{
			Choices: []chatChoice{
				{
					Index: 0,
					Message: Message{
						Role:    RoleAssistant,
						Content: responseText,
					},
					FinishReason: "stop",
				},
			},
			Model: req.Model,
			Usage: chatUsage{
				PromptTokens:     50,
				CompletionTokens: 30,
				TotalTokens:      80,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// TestChatCompletionBasic verifica uma chamada básica de chat completion.
func TestChatCompletionBasic(t *testing.T) {
	server := mockChatServer(t, "Olá! Como posso ajudar?", http.StatusOK)
	defer server.Close()

	llm := NewOpenAILLM("test-key", "gpt-4o-mini").WithURL(server.URL)

	messages := []Message{
		{Role: RoleSystem, Content: "Você é um assistente útil."},
		{Role: RoleUser, Content: "Olá"},
	}

	response, err := llm.ChatCompletion(context.Background(), messages)
	if err != nil {
		t.Fatalf("ChatCompletion falhou: %v", err)
	}

	if response != "Olá! Como posso ajudar?" {
		t.Errorf("response = %q, want %q", response, "Olá! Como posso ajudar?")
	}
}

// TestChatCompletionWithSystemPrompt verifica que o system prompt é enviado.
func TestChatCompletionWithSystemPrompt(t *testing.T) {
	var receivedMessages []Message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedMessages = req.Messages

		resp := chatResponse{
			Choices: []chatChoice{{
				Message: Message{Role: RoleAssistant, Content: "resposta"},
			}},
			Model: req.Model,
			Usage: chatUsage{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("key", "gpt-4o-mini").WithURL(server.URL)

	// RN-13: System prompt com instruções de comportamento.
	systemPrompt := "Responda apenas com base no contexto fornecido."
	messages := []Message{
		{Role: RoleSystem, Content: systemPrompt},
		{Role: RoleUser, Content: "Pergunta do usuário"},
	}

	llm.ChatCompletion(context.Background(), messages)

	if len(receivedMessages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(receivedMessages))
	}
	if receivedMessages[0].Role != RoleSystem {
		t.Errorf("messages[0].Role = %s, want system", receivedMessages[0].Role)
	}
	if receivedMessages[0].Content != systemPrompt {
		t.Errorf("system prompt não foi preservado")
	}
}

// TestChatCompletionEmptyMessages verifica que lista vazia retorna erro.
func TestChatCompletionEmptyMessages(t *testing.T) {
	llm := NewOpenAILLM("key", "gpt-4o-mini")

	_, err := llm.ChatCompletion(context.Background(), []Message{})
	if err == nil {
		t.Error("ChatCompletion deveria falhar para lista vazia")
	}
}

// TestChatCompletionAPIError verifica tratamento de erro 500.
func TestChatCompletionAPIError(t *testing.T) {
	server := mockChatServer(t, "", http.StatusInternalServerError)
	defer server.Close()

	llm := NewOpenAILLM("key", "gpt-4o-mini").WithURL(server.URL)

	_, err := llm.ChatCompletion(context.Background(), []Message{
		{Role: RoleUser, Content: "test"},
	})
	if err == nil {
		t.Error("ChatCompletion deveria falhar quando API retorna 500")
	}
}

// TestChatCompletionAPIRateLimit verifica tratamento de erro 429 (rate limit).
func TestChatCompletionAPIRateLimit(t *testing.T) {
	server := mockChatServer(t, "", http.StatusTooManyRequests)
	defer server.Close()

	llm := NewOpenAILLM("key", "gpt-4o-mini").WithURL(server.URL)

	_, err := llm.ChatCompletion(context.Background(), []Message{
		{Role: RoleUser, Content: "test"},
	})
	if err == nil {
		t.Error("ChatCompletion deveria falhar com rate limit")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("erro deveria mencionar status 429, got: %s", err.Error())
	}
}

// TestChatCompletionAuthorizationHeader verifica que o header Authorization é enviado.
func TestChatCompletionAuthorizationHeader(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")

		resp := chatResponse{
			Choices: []chatChoice{{Message: Message{Role: RoleAssistant, Content: "ok"}}},
			Model:   "gpt-4o-mini",
			Usage:   chatUsage{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("sk-my-secret-key", "gpt-4o-mini").WithURL(server.URL)
	llm.ChatCompletion(context.Background(), []Message{{Role: RoleUser, Content: "test"}})

	if receivedAuth != "Bearer sk-my-secret-key" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer sk-my-secret-key")
	}
}

// TestChatCompletionModelSentInRequest verifica que o modelo é enviado na request.
func TestChatCompletionModelSentInRequest(t *testing.T) {
	var receivedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedModel = req.Model

		resp := chatResponse{
			Choices: []chatChoice{{Message: Message{Role: RoleAssistant, Content: "ok"}}},
			Model:   req.Model,
			Usage:   chatUsage{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("key", "gpt-4o-mini").WithURL(server.URL)
	llm.ChatCompletion(context.Background(), []Message{{Role: RoleUser, Content: "test"}})

	if receivedModel != "gpt-4o-mini" {
		t.Errorf("model = %q, want %q", receivedModel, "gpt-4o-mini")
	}
}

// TestChatCompletionContextCancelled verifica que contexto cancelado interrompe.
func TestChatCompletionContextCancelled(t *testing.T) {
	server := mockChatServer(t, "resposta", http.StatusOK)
	defer server.Close()

	llm := NewOpenAILLM("key", "gpt-4o-mini").WithURL(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := llm.ChatCompletion(ctx, []Message{{Role: RoleUser, Content: "test"}})
	if err == nil {
		t.Error("ChatCompletion deveria falhar com contexto cancelado")
	}
}

// TestChatCompletionEmptyChoices verifica tratamento quando API retorna sem choices.
func TestChatCompletionEmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatResponse{
			Choices: []chatChoice{},
			Model:   "gpt-4o-mini",
			Usage:   chatUsage{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("key", "gpt-4o-mini").WithURL(server.URL)

	_, err := llm.ChatCompletion(context.Background(), []Message{
		{Role: RoleUser, Content: "test"},
	})
	if err == nil {
		t.Error("ChatCompletion deveria falhar quando não há choices")
	}
	if !strings.Contains(err.Error(), "sem choices") {
		t.Errorf("erro deveria mencionar 'sem choices', got: %s", err.Error())
	}
}

// TestChatCompletionMultipleMessages verifica que múltiplas mensagens são enviadas
// (simulando histórico de conversa — RN-18).
func TestChatCompletionMultipleMessages(t *testing.T) {
	var receivedCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedCount = len(req.Messages)

		resp := chatResponse{
			Choices: []chatChoice{{Message: Message{Role: RoleAssistant, Content: "resposta com contexto"}}},
			Model:   req.Model,
			Usage:   chatUsage{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llm := NewOpenAILLM("key", "gpt-4o-mini").WithURL(server.URL)

	messages := []Message{
		{Role: RoleSystem, Content: "System prompt"},
		{Role: RoleUser, Content: "Primeira pergunta"},
		{Role: RoleAssistant, Content: "Primeira resposta"},
		{Role: RoleUser, Content: "Follow-up pergunta"},
	}

	response, err := llm.ChatCompletion(context.Background(), messages)
	if err != nil {
		t.Fatalf("ChatCompletion falhou: %v", err)
	}

	if receivedCount != 4 {
		t.Errorf("API recebeu %d mensagens, want 4", receivedCount)
	}

	if response != "resposta com contexto" {
		t.Errorf("response = %q, want %q", response, "resposta com contexto")
	}
}
