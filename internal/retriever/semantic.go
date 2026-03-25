package retriever

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/renatomagalhaes/askwise/internal/embedding"
	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

// SemanticRetriever implementa Retriever usando embedding + busca vetorial.
// Combina Embedder e VectorStore para recuperar contexto relevante.
type SemanticRetriever struct {
	embedder   embedding.Embedder
	store      vectorstore.VectorStore
	collection string

	// ScoreThreshold é o score mínimo para um chunk ser considerado relevante.
	// RN-11: Padrão 0.5.
	ScoreThreshold float32
}

// NewSemanticRetriever cria um novo SemanticRetriever.
func NewSemanticRetriever(
	embedder embedding.Embedder,
	store vectorstore.VectorStore,
	collection string,
	scoreThreshold float32,
) *SemanticRetriever {
	return &SemanticRetriever{
		embedder:       embedder,
		store:          store,
		collection:     collection,
		ScoreThreshold: scoreThreshold,
	}
}

// Retrieve busca os topK chunks mais relevantes para a query.
// 1. Gera embedding da query
// 2. Busca no VectorStore
// 3. Filtra por score threshold (RN-11)
// 4. Aplica diversidade de fontes (RN-12)
func (r *SemanticRetriever) Retrieve(ctx context.Context, query string, topK int) ([]RetrievedChunk, error) {
	start := time.Now()

	// 1. Gera embedding da query.
	queryVec, err := r.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("falha ao gerar embedding da query: %w", err)
	}

	// Busca com margem extra para ter opções na etapa de diversidade.
	// RN-12: Se todos os top-K forem do mesmo documento, expande busca.
	searchLimit := topK * 2
	if searchLimit < 10 {
		searchLimit = 10
	}

	// 2. Busca no VectorStore.
	results, err := r.store.Search(ctx, r.collection, queryVec, searchLimit)
	if err != nil {
		return nil, fmt.Errorf("falha na busca vetorial: %w", err)
	}

	// 3. Filtra por score threshold (RN-11).
	filtered := r.filterByScore(results)

	// 4. Aplica diversidade de fontes (RN-12).
	diverse := r.applyDiversity(filtered, topK)

	chunks := make([]RetrievedChunk, len(diverse))
	for i, sr := range diverse {
		chunks[i] = searchResultToChunk(sr)
	}

	slog.Info("retrieval completed",
		"component", "retriever",
		"query_length", len(query),
		"search_results", len(results),
		"after_threshold", len(filtered),
		"returned", len(chunks),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return chunks, nil
}

// filterByScore remove resultados abaixo do score threshold.
// RN-11: Apenas chunks com score >= 0.5 devem ser usados.
func (r *SemanticRetriever) filterByScore(results []vectorstore.SearchResult) []vectorstore.SearchResult {
	var filtered []vectorstore.SearchResult
	for _, sr := range results {
		if sr.Score >= r.ScoreThreshold {
			filtered = append(filtered, sr)
		}
	}
	return filtered
}

// applyDiversity garante variedade de fontes nos resultados.
// RN-12: Preferir chunks de documentos diferentes quando possível.
//
// Estratégia: round-robin por documento. Seleciona o melhor chunk de cada
// documento primeiro, depois o segundo melhor, e assim por diante até
// atingir topK.
func (r *SemanticRetriever) applyDiversity(results []vectorstore.SearchResult, topK int) []vectorstore.SearchResult {
	if len(results) <= topK {
		return results
	}

	// Agrupa por document_id mantendo a ordem de score.
	byDoc := make(map[string][]vectorstore.SearchResult)
	var docOrder []string

	for _, sr := range results {
		docID := payloadString(sr.Payload, "document_id")
		if _, exists := byDoc[docID]; !exists {
			docOrder = append(docOrder, docID)
		}
		byDoc[docID] = append(byDoc[docID], sr)
	}

	// Se já temos diversidade natural (múltiplos docs), usa round-robin.
	// Se todos são do mesmo doc, simplesmente retorna os top-K.
	if len(docOrder) <= 1 {
		if len(results) > topK {
			return results[:topK]
		}
		return results
	}

	// Round-robin: pega 1 de cada documento por rodada.
	var diverse []vectorstore.SearchResult
	indices := make(map[string]int)

	for len(diverse) < topK {
		added := false
		for _, docID := range docOrder {
			if len(diverse) >= topK {
				break
			}
			idx := indices[docID]
			if idx < len(byDoc[docID]) {
				diverse = append(diverse, byDoc[docID][idx])
				indices[docID] = idx + 1
				added = true
			}
		}
		if !added {
			break
		}
	}

	return diverse
}

// searchResultToChunk converte um SearchResult do VectorStore em RetrievedChunk.
func searchResultToChunk(sr vectorstore.SearchResult) RetrievedChunk {
	return RetrievedChunk{
		Text:     payloadString(sr.Payload, "text"),
		FileName: payloadString(sr.Payload, "file_name"),
		FileType: payloadString(sr.Payload, "file_type"),
		Score:    sr.Score,
		Index:    payloadInt(sr.Payload, "chunk_index"),
	}
}

// payloadString extrai um valor string do payload do SearchResult.
func payloadString(payload map[string]any, key string) string {
	if v, ok := payload[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// payloadInt extrai um valor int do payload do SearchResult.
func payloadInt(payload map[string]any, key string) int {
	if v, ok := payload[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case float64:
			return int(n)
		case int64:
			return int(n)
		}
	}
	return 0
}
