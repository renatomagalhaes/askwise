package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockEmbeddingServer cria um servidor HTTP mock que simula a API de embeddings da OpenAI.
// Retorna vetores de dimensão dim para cada texto recebido.
func mockEmbeddingServer(t *testing.T, dim int, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifica headers obrigatórios.
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

		var req embeddingRequest
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

		// Gera embeddings fake (vetor de zeros com dim dimensões).
		data := make([]embeddingData, len(req.Input))
		for i := range req.Input {
			vec := make([]float32, dim)
			for j := range vec {
				vec[j] = float32(i+1) * 0.001
			}
			data[i] = embeddingData{
				Embedding: vec,
				Index:     i,
			}
		}

		resp := embeddingResponse{
			Data:  data,
			Model: req.Model,
			Usage: embeddingUsage{
				PromptTokens: len(req.Input) * 5,
				TotalTokens:  len(req.Input) * 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// TestEmbedSingleText verifica embedding de um único texto.
func TestEmbedSingleText(t *testing.T) {
	server := mockEmbeddingServer(t, 1536, http.StatusOK)
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small").WithURL(server.URL)

	embeddings, err := embedder.Embed(context.Background(), []string{"hello world"})
	if err != nil {
		t.Fatalf("Embed falhou: %v", err)
	}

	if len(embeddings) != 1 {
		t.Fatalf("len(embeddings) = %d, want 1", len(embeddings))
	}

	if len(embeddings[0]) != 1536 {
		t.Errorf("dimensões = %d, want 1536", len(embeddings[0]))
	}
}

// TestEmbedMultipleTexts verifica embedding de múltiplos textos em um batch.
func TestEmbedMultipleTexts(t *testing.T) {
	server := mockEmbeddingServer(t, 1536, http.StatusOK)
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small").WithURL(server.URL)

	texts := []string{"primeiro texto", "segundo texto", "terceiro texto"}
	embeddings, err := embedder.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("Embed falhou: %v", err)
	}

	if len(embeddings) != 3 {
		t.Fatalf("len(embeddings) = %d, want 3", len(embeddings))
	}

	// Cada embedding deve ter 1536 dimensões.
	for i, emb := range embeddings {
		if len(emb) != 1536 {
			t.Errorf("embeddings[%d] dimensões = %d, want 1536", i, len(emb))
		}
	}
}

// TestEmbedEmptyList verifica que lista vazia retorna erro.
func TestEmbedEmptyList(t *testing.T) {
	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small")

	_, err := embedder.Embed(context.Background(), []string{})
	if err == nil {
		t.Error("Embed deveria falhar para lista vazia")
	}
}

// TestEmbedQuerySingle verifica EmbedQuery para uma query.
func TestEmbedQuerySingle(t *testing.T) {
	server := mockEmbeddingServer(t, 1536, http.StatusOK)
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small").WithURL(server.URL)

	vec, err := embedder.EmbedQuery(context.Background(), "como resolver erro 5032?")
	if err != nil {
		t.Fatalf("EmbedQuery falhou: %v", err)
	}

	if len(vec) != 1536 {
		t.Errorf("dimensões = %d, want 1536", len(vec))
	}
}

// TestEmbedQueryEmpty verifica que query vazia retorna erro.
func TestEmbedQueryEmpty(t *testing.T) {
	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small")

	_, err := embedder.EmbedQuery(context.Background(), "")
	if err == nil {
		t.Error("EmbedQuery deveria falhar para query vazia")
	}
}

// TestEmbedAPIError verifica tratamento de erros da API.
func TestEmbedAPIError(t *testing.T) {
	server := mockEmbeddingServer(t, 1536, http.StatusInternalServerError)
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small").WithURL(server.URL)

	_, err := embedder.Embed(context.Background(), []string{"test"})
	if err == nil {
		t.Error("Embed deveria falhar quando API retorna 500")
	}
}

// TestEmbedAPIUnauthorized verifica tratamento de erro 401 (sem API key).
func TestEmbedAPIUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "Incorrect API key provided"},
		})
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("invalid-key", "text-embedding-3-small").WithURL(server.URL)

	_, err := embedder.Embed(context.Background(), []string{"test"})
	if err == nil {
		t.Error("Embed deveria falhar com API key inválida")
	}
}

// TestEmbedContextCancelled verifica que contexto cancelado interrompe a chamada.
func TestEmbedContextCancelled(t *testing.T) {
	server := mockEmbeddingServer(t, 1536, http.StatusOK)
	defer server.Close()

	embedder := NewOpenAIEmbedder("test-key", "text-embedding-3-small").WithURL(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := embedder.Embed(ctx, []string{"test"})
	if err == nil {
		t.Error("Embed deveria falhar com contexto cancelado")
	}
}

// TestEmbedAuthorizationHeader verifica que o header Authorization é enviado.
func TestEmbedAuthorizationHeader(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")

		resp := embeddingResponse{
			Data:  []embeddingData{{Embedding: make([]float32, 4), Index: 0}},
			Model: "test",
			Usage: embeddingUsage{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("sk-test-key-123", "text-embedding-3-small").WithURL(server.URL)
	embedder.Embed(context.Background(), []string{"test"})

	if receivedAuth != "Bearer sk-test-key-123" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer sk-test-key-123")
	}
}

// TestEmbedModelSentInRequest verifica que o modelo é enviado na request.
func TestEmbedModelSentInRequest(t *testing.T) {
	var receivedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embeddingRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedModel = req.Model

		resp := embeddingResponse{
			Data:  []embeddingData{{Embedding: make([]float32, 4), Index: 0}},
			Model: req.Model,
			Usage: embeddingUsage{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	embedder := NewOpenAIEmbedder("key", "text-embedding-3-small").WithURL(server.URL)
	embedder.Embed(context.Background(), []string{"test"})

	if receivedModel != "text-embedding-3-small" {
		t.Errorf("model = %q, want %q", receivedModel, "text-embedding-3-small")
	}
}
