package embedding

import "context"

// Embedder define a interface para geração de embeddings vetoriais.
// Spec: design/01-ARQUITETURA.md §2.5
//
// Cada implementação encapsula a comunicação com um provider de IA.
// A interface permite mock nos testes e troca de provider sem impacto no pipeline.
type Embedder interface {
	// Embed gera embeddings para uma lista de textos.
	// Retorna um slice de vetores na mesma ordem dos textos de entrada.
	// RF-04: Processar múltiplos chunks em uma única chamada quando possível.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedQuery gera embedding para uma única query de busca.
	// Conveniência sobre Embed para o caso comum de uma query.
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
}
