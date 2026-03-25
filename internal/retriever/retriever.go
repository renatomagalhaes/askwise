package retriever

import "context"

// RetrievedChunk representa um chunk recuperado e pronto para uso no prompt da LLM.
// Spec: design/02-MODELO-DADOS.md §4 (RetrievedChunk)
type RetrievedChunk struct {
	// Text é o conteúdo textual do chunk.
	Text string

	// FileName é o nome do arquivo fonte (usado na citação de fontes, RN-14).
	FileName string

	// FileType é o tipo do arquivo fonte (ex: "md", "pdf").
	FileType string

	// Score é a similaridade coseno com a query (0.0 a 1.0).
	Score float32

	// Index é a posição do chunk dentro do documento original.
	Index int
}

// Retriever define a interface para recuperação de contexto relevante.
// Spec: design/01-ARQUITETURA.md §2.7
//
// Cada implementação encapsula a estratégia de busca. A interface permite
// trocar de busca semântica para BM25, híbrida, etc.
type Retriever interface {
	// Retrieve busca os topK chunks mais relevantes para a query.
	// Aplica score threshold e diversidade de fontes.
	Retrieve(ctx context.Context, query string, topK int) ([]RetrievedChunk, error)
}
