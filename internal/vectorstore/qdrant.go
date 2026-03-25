package vectorstore

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

// QdrantStore implementa VectorStore usando a REST API do Qdrant.
// ADR-004: Usa REST API direta (net/http) para manter simplicidade (RNF-03).
type QdrantStore struct {
	baseURL string
	client  *http.Client
}

// NewQdrant cria uma nova instância de QdrantStore.
// O addr deve ser no formato "host:port" (ex: "localhost:6333").
func NewQdrant(addr string) *QdrantStore {
	return &QdrantStore{
		baseURL: fmt.Sprintf("http://%s", addr),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EnsureCollection cria a collection se ela não existir.
// Se já existir, a API retorna sucesso (operação idempotente).
// Spec: design/02-MODELO-DADOS.md §3 (Collection config: size=1536, distance=Cosine)
func (q *QdrantStore) EnsureCollection(ctx context.Context, name string, dimension int) error {
	// Primeiro verifica se a collection já existe
	url := fmt.Sprintf("%s/collections/%s", q.baseURL, name)
	resp, err := q.doRequest(ctx, http.MethodGet, url, nil)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		slog.Debug("collection already exists", "component", "vectorstore", "collection", name)
		return nil
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Cria a collection com a configuração de vetores
	body := map[string]any{
		"vectors": map[string]any{
			"size":     dimension,
			"distance": "Cosine",
		},
	}

	url = fmt.Sprintf("%s/collections/%s", q.baseURL, name)
	resp, err = q.doRequest(ctx, http.MethodPut, url, body)
	if err != nil {
		return fmt.Errorf("falha ao criar collection %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return q.readError(resp, "criar collection "+name)
	}

	slog.Info("collection created", "component", "vectorstore", "collection", name, "dimension", dimension)
	return nil
}

// Upsert insere ou atualiza pontos na collection.
// Spec: cada ponto contém vetor + payload com metadados (RN-08).
func (q *QdrantStore) Upsert(ctx context.Context, collection string, points []Point) error {
	// Converte os pontos para o formato esperado pela API do Qdrant
	qdrantPoints := make([]map[string]any, len(points))
	for i, p := range points {
		qdrantPoints[i] = map[string]any{
			"id":      p.ID,
			"vector":  p.Vector,
			"payload": p.Payload,
		}
	}

	body := map[string]any{
		"points": qdrantPoints,
	}

	url := fmt.Sprintf("%s/collections/%s/points", q.baseURL, collection)
	resp, err := q.doRequest(ctx, http.MethodPut, url, body)
	if err != nil {
		return fmt.Errorf("falha no upsert de %d pontos: %w", len(points), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return q.readError(resp, "upsert")
	}

	slog.Debug("points upserted", "component", "vectorstore", "collection", collection, "count", len(points))
	return nil
}

// Search busca os topK pontos mais similares ao vetor fornecido.
// Retorna resultados ordenados por score decrescente (mais similar primeiro).
func (q *QdrantStore) Search(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error) {
	body := map[string]any{
		"vector":       vector,
		"limit":        topK,
		"with_payload": true,
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, collection)
	resp, err := q.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("falha na busca vetorial: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, q.readError(resp, "search")
	}

	// Decodifica a resposta do Qdrant
	var result struct {
		Result []struct {
			ID      any            `json:"id"`
			Score   float32        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("falha ao decodificar resposta de search: %w", err)
	}

	// Converte para nosso tipo SearchResult
	results := make([]SearchResult, len(result.Result))
	for i, r := range result.Result {
		results[i] = SearchResult{
			ID:      fmt.Sprintf("%v", r.ID),
			Score:   r.Score,
			Payload: r.Payload,
		}
	}

	slog.Debug("search completed", "component", "vectorstore", "collection", collection, "results", len(results))
	return results, nil
}

// DeleteByDocID remove todos os pontos cujo payload.document_id corresponde ao docID.
// RN-21: Garante que ao deletar um documento, seus chunks são removidos do vectorstore.
func (q *QdrantStore) DeleteByDocID(ctx context.Context, collection string, docID string) error {
	body := map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{
					"key":   "document_id",
					"match": map[string]any{"value": docID},
				},
			},
		},
	}

	url := fmt.Sprintf("%s/collections/%s/points/delete", q.baseURL, collection)
	resp, err := q.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("falha ao deletar pontos do doc %s: %w", docID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return q.readError(resp, "delete por doc_id")
	}

	slog.Info("points deleted", "component", "vectorstore", "collection", collection, "document_id", docID)
	return nil
}

// --- Helpers internos ---

// doRequest executa uma request HTTP contra a API do Qdrant.
// Serializa o body como JSON e adiciona os headers necessários.
func (q *QdrantStore) doRequest(ctx context.Context, method, url string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("falha ao serializar body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return q.client.Do(req)
}

// readError lê o corpo de uma resposta de erro do Qdrant e retorna um erro formatado.
func (q *QdrantStore) readError(resp *http.Response, operation string) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("qdrant %s falhou (HTTP %d): %s", operation, resp.StatusCode, string(body))
}
