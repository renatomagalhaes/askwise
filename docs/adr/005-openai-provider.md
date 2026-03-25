# ADR-005: OpenAI como Provider de Embeddings e LLM

## Status

Aceita

## Contexto

O pipeline RAG precisa de dois serviços de IA:
1. **Embeddings**: Converter texto em vetores numéricos
2. **LLM (Chat Completion)**: Gerar respostas baseadas em contexto

Precisamos escolher o provider.

Opções consideradas:

| Opção              | Embeddings | LLM  | Custo      | Complexidade |
|--------------------|------------|------|------------|--------------|
| OpenAI API         | Sim        | Sim  | Pay-per-use| Baixa        |
| Ollama (local)     | Sim        | Sim  | Grátis     | Média        |
| AWS Bedrock        | Sim        | Sim  | Pay-per-use| Alta         |
| Google Vertex AI   | Sim        | Sim  | Pay-per-use| Alta         |
| HuggingFace local  | Sim        | Sim  | Grátis     | Alta (GPU)   |

## Decisão

Adotamos **OpenAI API** com os modelos:
- Embeddings: `text-embedding-3-small` (1536 dimensões, $0.02/1M tokens)
- LLM: `gpt-4o-mini` (rápido, $0.15/1M input tokens)

## Justificativa

- **Simplicidade**: Uma API key, dois endpoints, pronto
- **Qualidade**: Modelos de referência no mercado
- **Custo baixo**: Para PoC, o custo estimado é ~$0.50/mês
- **Documentação**: Extensa e com exemplos em múltiplas linguagens
- **Sem GPU**: Não precisa de hardware local especializado
- **Foco no RAG**: Energia gasta aprendendo RAG, não configurando infra de IA

## Interfaces para Extensibilidade

As interfaces `Embedder` e `LLM` permitem trocar o provider sem mudar o código:

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
}

type LLM interface {
    ChatCompletion(ctx context.Context, messages []Message) (string, error)
}
```

No futuro, pode-se implementar `OllamaEmbedder` ou `OllamaLLM` sem alterar o
restante do código.

## Consequências

### Positivas
- Setup em 1 minuto (criar API key no site da OpenAI)
- Qualidade de resposta alta desde o primeiro teste
- Custo desprezível para volume de PoC

### Negativas
- Requer internet (não funciona offline)
- Requer API key (custo, mesmo que baixo)
- Dependência de serviço externo (latência, disponibilidade)
- Dados enviados para servidores da OpenAI (considerar privacidade)

### Mitigação para Futuro
- Implementar OllamaEmbedder + OllamaLLM para uso offline/grátis
- Configurável via variável de ambiente: `AI_PROVIDER=openai|ollama`
