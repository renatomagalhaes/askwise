package retriever

import (
	"context"
	"fmt"
	"testing"

	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

// --- Mocks ---

// mockEmbedder simula a interface Embedder para testes.
type mockEmbedder struct {
	embedFn      func(ctx context.Context, texts []string) ([][]float32, error)
	embedQueryFn func(ctx context.Context, query string) ([]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = make([]float32, 4)
	}
	return vecs, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if m.embedQueryFn != nil {
		return m.embedQueryFn(ctx, query)
	}
	return make([]float32, 4), nil
}

// mockVectorStore simula a interface VectorStore para testes.
type mockVectorStore struct {
	searchFn func(ctx context.Context, collection string, vector []float32, topK int) ([]vectorstore.SearchResult, error)
}

func (m *mockVectorStore) EnsureCollection(ctx context.Context, name string, dimension int) error {
	return nil
}
func (m *mockVectorStore) Upsert(ctx context.Context, collection string, points []vectorstore.Point) error {
	return nil
}
func (m *mockVectorStore) Search(ctx context.Context, collection string, vector []float32, topK int) ([]vectorstore.SearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, collection, vector, topK)
	}
	return nil, nil
}
func (m *mockVectorStore) DeleteByDocID(ctx context.Context, collection string, docID string) error {
	return nil
}

// --- Helpers ---

func makeResult(docID string, score float32, chunkIndex int) vectorstore.SearchResult {
	return vectorstore.SearchResult{
		ID:    fmt.Sprintf("chunk-%s-%d", docID, chunkIndex),
		Score: score,
		Payload: map[string]any{
			"document_id": docID,
			"chunk_index": chunkIndex,
			"text":        fmt.Sprintf("Texto do chunk %d do doc %s", chunkIndex, docID),
			"file_name":   docID + ".md",
			"file_type":   "md",
		},
	}
}

// --- Testes ---

// TestRetrieveBasic verifica o fluxo básico de retrieve.
func TestRetrieveBasic(t *testing.T) {
	embedder := &mockEmbedder{}
	store := &mockVectorStore{
		searchFn: func(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
			return []vectorstore.SearchResult{
				makeResult("doc1", 0.9, 0),
				makeResult("doc1", 0.8, 1),
				makeResult("doc2", 0.7, 0),
			}, nil
		},
	}

	ret := NewSemanticRetriever(embedder, store, "askwise", 0.5)
	chunks, err := ret.Retrieve(context.Background(), "como resolver erro?", 5)
	if err != nil {
		t.Fatalf("Retrieve falhou: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("len(chunks) = %d, want 3", len(chunks))
	}

	if chunks[0].Score != 0.9 {
		t.Errorf("chunks[0].Score = %f, want 0.9", chunks[0].Score)
	}
}

// TestRetrieveScoreThreshold verifica que chunks abaixo do threshold são filtrados.
// RN-11: Score mínimo de 0.5.
func TestRetrieveScoreThreshold(t *testing.T) {
	store := &mockVectorStore{
		searchFn: func(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
			return []vectorstore.SearchResult{
				makeResult("doc1", 0.9, 0),
				makeResult("doc1", 0.6, 1),
				makeResult("doc2", 0.4, 0), // abaixo do threshold
				makeResult("doc3", 0.3, 0), // abaixo do threshold
			}, nil
		},
	}

	ret := NewSemanticRetriever(&mockEmbedder{}, store, "askwise", 0.5)
	chunks, err := ret.Retrieve(context.Background(), "query", 5)
	if err != nil {
		t.Fatalf("Retrieve falhou: %v", err)
	}

	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2 (threshold 0.5 filtra 2)", len(chunks))
	}
}

// TestRetrieveDiversity verifica que resultados de documentos diferentes são preferidos.
// RN-12: Diversidade de fontes.
func TestRetrieveDiversity(t *testing.T) {
	store := &mockVectorStore{
		searchFn: func(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
			return []vectorstore.SearchResult{
				makeResult("doc1", 0.95, 0),
				makeResult("doc1", 0.93, 1),
				makeResult("doc1", 0.90, 2),
				makeResult("doc2", 0.88, 0),
				makeResult("doc2", 0.85, 1),
				makeResult("doc3", 0.80, 0),
			}, nil
		},
	}

	ret := NewSemanticRetriever(&mockEmbedder{}, store, "askwise", 0.5)
	chunks, err := ret.Retrieve(context.Background(), "query", 3)
	if err != nil {
		t.Fatalf("Retrieve falhou: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("len(chunks) = %d, want 3", len(chunks))
	}

	// Com diversidade, deve ter chunks de docs diferentes.
	docs := make(map[string]bool)
	for _, c := range chunks {
		docs[c.FileName] = true
	}

	if len(docs) < 2 {
		t.Errorf("Diversidade: apenas %d doc(s) nos resultados, esperava >= 2", len(docs))
	}
}

// TestRetrieveSingleDocFallback verifica que quando todos os chunks são do mesmo doc,
// retorna os top-K sem erro.
func TestRetrieveSingleDocFallback(t *testing.T) {
	store := &mockVectorStore{
		searchFn: func(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
			return []vectorstore.SearchResult{
				makeResult("doc1", 0.95, 0),
				makeResult("doc1", 0.90, 1),
				makeResult("doc1", 0.85, 2),
				makeResult("doc1", 0.80, 3),
				makeResult("doc1", 0.75, 4),
			}, nil
		},
	}

	ret := NewSemanticRetriever(&mockEmbedder{}, store, "askwise", 0.5)
	chunks, err := ret.Retrieve(context.Background(), "query", 3)
	if err != nil {
		t.Fatalf("Retrieve falhou: %v", err)
	}

	if len(chunks) != 3 {
		t.Fatalf("len(chunks) = %d, want 3", len(chunks))
	}
}

// TestRetrieveNoResults verifica que busca sem resultados retorna slice vazio.
// RN-17: Se nenhum chunk relevante for encontrado, informar ao usuário.
func TestRetrieveNoResults(t *testing.T) {
	store := &mockVectorStore{
		searchFn: func(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
			return []vectorstore.SearchResult{}, nil
		},
	}

	ret := NewSemanticRetriever(&mockEmbedder{}, store, "askwise", 0.5)
	chunks, err := ret.Retrieve(context.Background(), "query obscura", 5)
	if err != nil {
		t.Fatalf("Retrieve falhou: %v", err)
	}

	if len(chunks) != 0 {
		t.Errorf("len(chunks) = %d, want 0", len(chunks))
	}
}

// TestRetrieveEmbedError verifica tratamento de erro no embedding.
func TestRetrieveEmbedError(t *testing.T) {
	embedder := &mockEmbedder{
		embedQueryFn: func(_ context.Context, _ string) ([]float32, error) {
			return nil, fmt.Errorf("API indisponível")
		},
	}

	ret := NewSemanticRetriever(embedder, &mockVectorStore{}, "askwise", 0.5)
	_, err := ret.Retrieve(context.Background(), "query", 5)
	if err == nil {
		t.Error("Retrieve deveria falhar quando embedding falha")
	}
}

// TestRetrieveSearchError verifica tratamento de erro na busca vetorial.
func TestRetrieveSearchError(t *testing.T) {
	store := &mockVectorStore{
		searchFn: func(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
			return nil, fmt.Errorf("Qdrant offline")
		},
	}

	ret := NewSemanticRetriever(&mockEmbedder{}, store, "askwise", 0.5)
	_, err := ret.Retrieve(context.Background(), "query", 5)
	if err == nil {
		t.Error("Retrieve deveria falhar quando busca vetorial falha")
	}
}

// TestRetrieveChunkFields verifica que todos os campos do RetrievedChunk são populados.
func TestRetrieveChunkFields(t *testing.T) {
	store := &mockVectorStore{
		searchFn: func(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
			return []vectorstore.SearchResult{
				{
					ID:    "point-1",
					Score: 0.85,
					Payload: map[string]any{
						"document_id": "doc-abc",
						"chunk_index": 3,
						"text":        "Conteúdo do chunk relevante",
						"file_name":   "manual.pdf",
						"file_type":   "pdf",
					},
				},
			}, nil
		},
	}

	ret := NewSemanticRetriever(&mockEmbedder{}, store, "askwise", 0.5)
	chunks, err := ret.Retrieve(context.Background(), "query", 5)
	if err != nil {
		t.Fatalf("Retrieve falhou: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}

	c := chunks[0]
	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"Text", c.Text, "Conteúdo do chunk relevante"},
		{"FileName", c.FileName, "manual.pdf"},
		{"FileType", c.FileType, "pdf"},
		{"Score", c.Score, float32(0.85)},
		{"Index", c.Index, 3},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}
}
