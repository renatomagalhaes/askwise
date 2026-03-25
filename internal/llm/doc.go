// Package llm encapsula a comunicação com modelos de linguagem (LLM) para chat completion.
//
// RF-07: O sistema deve enviar prompt para LLM e retornar resposta.
// Spec dirigindo: RF-07 (pipeline RAG), RN-13 (resposta baseada em contexto),
// RN-14 (citação de fontes), RN-15 (idioma), RN-16 (tom profissional).
// ADR-005: OpenAI como provider de IA.
//
// A interface LLM permite trocar o provider (OpenAI → Ollama, etc.)
// sem alterar o restante do pipeline. A implementação padrão usa a API
// de chat completion da OpenAI com o modelo gpt-4o-mini.
//
// O system prompt é configurável e deve incluir as instruções de
// comportamento definidas nas regras de negócio (RN-13 a RN-16).
package llm
