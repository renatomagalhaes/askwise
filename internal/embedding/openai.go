package embedding

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
	// defaultEmbeddingURL é o endpoint da API de embeddings da OpenAI.
	defaultEmbeddingURL = "https://api.openai.com/v1/embeddings"

	// maxBatchSize é o limite de textos por chamada à API (RF-04).
	maxBatchSize = 100
)

// OpenAIEmbedder implementa Embedder usando a API de embeddings da OpenAI.
// ADR-005: OpenAI como provider de embeddings.
type OpenAIEmbedder struct {
	apiKey string
	model  string
	url    string
	client *http.Client
}

// NewOpenAIEmbedder cria um novo OpenAIEmbedder.
// apiKey é a chave da API da OpenAI (OPENAI_API_KEY).
// model é o modelo de embedding (ex: "text-embedding-3-small").
func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		apiKey: apiKey,
		model:  model,
		url:    defaultEmbeddingURL,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// WithURL substitui a URL da API (usado em testes com mock server).
func (e *OpenAIEmbedder) WithURL(url string) *OpenAIEmbedder {
	e.url = url
	return e
}

// WithHTTPClient substitui o http.Client (usado em testes).
func (e *OpenAIEmbedder) WithHTTPClient(client *http.Client) *OpenAIEmbedder {
	e.client = client
	return e
}

// Embed gera embeddings para uma lista de textos.
// Se a lista exceder maxBatchSize, divide em batches automáticos.
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("lista de textos vazia")
	}

	start := time.Now()

	// Divide em batches de até maxBatchSize textos (RF-04).
	var allEmbeddings [][]float32
	for i := 0; i < len(texts); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]

		embeddings, err := e.embedBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("falha no batch %d-%d: %w", i, end, err)
		}
		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	slog.Info("embeddings generated",
		"component", "embedding",
		"texts", len(texts),
		"dimensions", len(allEmbeddings[0]),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return allEmbeddings, nil
}

// EmbedQuery gera embedding para uma única query de busca.
func (e *OpenAIEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if query == "" {
		return nil, fmt.Errorf("query vazia")
	}

	embeddings, err := e.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar embedding da query: %w", err)
	}

	return embeddings[0], nil
}

// embeddingRequest é o corpo da requisição para a API de embeddings.
type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// embeddingResponse é a resposta da API de embeddings.
type embeddingResponse struct {
	Data  []embeddingData `json:"data"`
	Model string          `json:"model"`
	Usage embeddingUsage  `json:"usage"`
}

type embeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type embeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// openAIErrorResponse captura erros da API da OpenAI.
type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// embedBatch faz uma chamada à API para um batch de textos.
func (e *OpenAIEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := embeddingRequest{
		Model: e.model,
		Input: texts,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("falha ao serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("falha ao criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha na chamada à API de embeddings: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler resposta da API: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openAIErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("API de embeddings retornou %d: %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API de embeddings retornou %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("falha ao parsear resposta da API: %w", err)
	}

	if len(embResp.Data) != len(texts) {
		return nil, fmt.Errorf("API retornou %d embeddings, esperava %d", len(embResp.Data), len(texts))
	}

	// Ordena pelo index para garantir alinhamento com os textos de entrada.
	embeddings := make([][]float32, len(texts))
	for _, d := range embResp.Data {
		if d.Index < 0 || d.Index >= len(texts) {
			return nil, fmt.Errorf("índice inválido na resposta: %d", d.Index)
		}
		embeddings[d.Index] = d.Embedding
	}

	slog.Debug("embedding batch completed",
		"component", "embedding",
		"batch_size", len(texts),
		"tokens_used", embResp.Usage.TotalTokens,
	)

	return embeddings, nil
}
