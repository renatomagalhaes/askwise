// Package rag é o orquestrador do pipeline RAG (Retrieval-Augmented Generation).
//
// RF-07: O sistema deve implementar o pipeline completo de RAG.
// Spec dirigindo: RF-01 a RF-07, RN-04 (duplicatas), RN-05 (sanitização),
// RN-08 (metadados), RN-10 a RN-17, design/01-ARQUITETURA.md §2.9, §3.
//
// O orquestrador conecta todos os componentes:
//
//	Pipeline de Ingestão:
//	  Arquivo → Parser → Texto → Chunker → Embed → VectorStore + Storage
//
//	Pipeline de Consulta:
//	  Pergunta → Retriever → Contexto → LLM → Resposta com fontes
//
// Cada componente é injetado via interface, permitindo mock nos testes
// e troca de implementação sem alterar a lógica de orquestração.
package rag
