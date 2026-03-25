// Package retriever combina embeddings e busca vetorial para encontrar
// contexto relevante a partir de uma pergunta do usuário.
//
// RF-07: Pipeline RAG — Retrieval: buscar os top-K chunks mais relevantes.
// Spec dirigindo: RN-10 (top-K=5), RN-11 (score mínimo 0.5),
// RN-12 (diversidade de fontes), design/01-ARQUITETURA.md §2.7.
//
// O SemanticRetriever recebe uma query textual, gera seu embedding,
// busca no VectorStore os chunks mais similares, filtra por score e
// aplica diversidade de fontes antes de retornar os resultados.
package retriever
