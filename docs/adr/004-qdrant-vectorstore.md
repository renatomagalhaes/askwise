# ADR-004: Qdrant como Vector Store

## Status

Aceita

## Contexto

O pipeline RAG precisa de um banco de dados vetorial para armazenar embeddings e
executar busca por similaridade (cosine similarity). Precisamos escolher qual usar.

Opções consideradas:

| Opção         | Prós                                    | Contras                              |
|---------------|-----------------------------------------|--------------------------------------|
| Qdrant        | REST API, client Go, Docker, grátis     | Serviço externo (Docker)             |
| Pgvector      | Integrado ao PostgreSQL                 | Precisa de PostgreSQL                |
| Weaviate      | Rico em features                        | Complexo, pesado                     |
| Milvus        | Escalável, features enterprise          | Pesado, complexo para PoC            |
| Pinecone      | Managed, zero-ops                       | Pago, vendor lock-in, sem self-host  |
| chromem-go    | Embedded em Go, sem servidor            | Imaturo, sem comunidade grande       |

## Decisão

Adotamos **Qdrant** rodando via Docker Compose.

## Justificativa

- **Client Go oficial**: `github.com/qdrant/go-client` com API gRPC
- **REST API**: Também expõe REST para debug e exploração manual
- **Docker simples**: `docker compose up -d` e está rodando
- **Grátis e open-source**: Apache 2.0 license
- **Documentação excelente**: Boa para aprendizado
- **Performance**: Rápido o suficiente para PoC e produção leve

## Consequências

### Positivas
- Setup via Docker Compose (1 comando)
- Dashboard web na porta 6333 para inspecionar dados
- API bem documentada, fácil de aprender
- Suporta filtros por payload (metadados) além de busca vetorial

### Negativas
- Serviço externo (precisa de Docker rodando)
- Porta 6333/6334 precisa estar livre
- Dados persistidos em volume Docker (qdrant_data)
