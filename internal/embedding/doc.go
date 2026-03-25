// Package embedding gera vetores numéricos (embeddings) a partir de texto.
//
// RF-04: O sistema deve gerar embeddings vetoriais para cada chunk.
// Spec dirigindo: RF-04 (geração de embeddings), design/01-ARQUITETURA.md §2.5.
// ADR-005: OpenAI como provider de IA.
//
// A interface Embedder permite trocar o provider (OpenAI → Ollama, etc.)
// sem alterar o restante do pipeline. A implementação padrão usa a API
// da OpenAI com o modelo text-embedding-3-small (1536 dimensões).
//
// Batch: a API aceita até 100 textos por chamada. O client divide
// automaticamente listas maiores em batches.
package embedding
