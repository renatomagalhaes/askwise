# AGENTS.md — Instruções para Agentes de IA

## Sobre o Projeto

AskWise é uma PoC de chatbot RAG (Retrieval-Augmented Generation) em Go.
O objetivo é educativo — aprender RAG e a metodologia SDD (Spec-Design-Development).

## Metodologia SDD

Este projeto segue a metodologia **SDD**. Antes de implementar qualquer código:

1. **Leia a Spec** em `docs/spec/` para entender O QUE construir
2. **Leia o Design** em `docs/design/` para entender COMO construir
3. **Implemente** seguindo as definições de spec e design

Se algo não está definido na spec ou design, pergunte ao usuário antes de inventar.

## Documentação Obrigatória

Antes de iniciar qualquer desenvolvimento, leia estes documentos na ordem:

1. `docs/spec/01-VISAO-GERAL.md` — Propósito e escopo
2. `docs/spec/02-CENARIO-NEGOCIO.md` — Contexto e personas
3. `docs/spec/03-REQUISITOS.md` — Requisitos funcionais e não-funcionais
4. `docs/spec/04-REGRAS-NEGOCIO.md` — Regras que o código deve seguir
5. `docs/design/01-ARQUITETURA.md` — Componentes e arquitetura
6. `docs/design/02-MODELO-DADOS.md` — Modelo de dados e structs
7. `docs/design/03-API-DESIGN.md` — Endpoints e contratos

## Estrutura do Código

```
cmd/
├── server/main.go        — API HTTP (upload de documentos)
└── chat/main.go          — CLI interativo (chat terminal)

internal/
├── document/             — Parsing de arquivos (PDF, CSV, TXT, YAML, JSON, MD)
├── chunker/              — Divisão de texto em chunks
├── embedding/            — Geração de embeddings via OpenAI
├── vectorstore/          — Client Qdrant (armazenar e buscar vetores)
├── retriever/            — Busca de contexto relevante (embedding + search)
├── llm/                  — Chat completion via OpenAI
├── rag/                  — Orquestrador do pipeline (conecta tudo)
└── storage/              — SQLite (metadados de documentos)
```

## Convenções de Código

### Linguagem e Estilo
- **Go 1.22+** com modules
- Usar standard library sempre que possível
- Interfaces para todos os componentes principais (testabilidade)
- Erros com contexto: `fmt.Errorf("falha ao processar %s: %w", filename, err)`
- Nomes descritivos em inglês para código, comentários podem ser em português

### Comentários
- Este é um projeto **educativo** — comente generosamente
- Cada pacote deve ter `doc.go` explicando a responsabilidade
- Funções públicas devem ter godoc
- Comentários devem explicar o **porquê**, não apenas o **o quê**
- Referenciar requisitos e regras: `// RF-01: Validação de formato` ou `// RN-06: Chunk size`

### Configuração
- Tudo via variáveis de ambiente (arquivo `.env`)
- Usar `.env.example` como template
- Nunca hardcodar API keys ou secrets

### Tratamento de Erros
- Sempre propagar erros com contexto
- Logs estruturados nas bordas (handlers HTTP, CLI)
- Não usar panic para erros de negócio

## Referência de Regras de Negócio

Ao implementar, referencie as regras em `docs/spec/04-REGRAS-NEGOCIO.md`:

| Componente       | Regras Aplicáveis                                  |
|------------------|----------------------------------------------------|
| Upload handler   | RN-01 (formato), RN-02 (tamanho), RN-03 (vazio)   |
| Document parser  | RN-03 (conteúdo extraível)                          |
| Chunker          | RN-06 (tamanho), RN-07 (overlap), RN-09 (formato) |
| Chunk metadata   | RN-08 (metadados obrigatórios)                      |
| Retriever        | RN-10 (top-K), RN-11 (score mínimo), RN-12 (diversidade) |
| LLM prompt       | RN-13 (baseado em contexto), RN-14 (fontes), RN-15 (idioma), RN-16 (tom) |
| Chat CLI         | RN-18 (histórico), RN-19 (comandos), RN-20 (feedback visual) |
| Delete handler   | RN-21 (integridade referencial)                     |
| Re-upload        | RN-04 (duplicatas), RN-22 (idempotência)            |

## Ordem de Implementação

Siga esta ordem para implementar os componentes:

```
1. internal/storage        — Base: SQLite para metadados
2. internal/embedding      — Integração OpenAI embeddings
3. internal/vectorstore    — Client Qdrant
4. internal/document       — Parsers de arquivos
5. internal/chunker        — Divisão de texto
6. internal/retriever      — Busca de contexto
7. internal/llm            — Chat completion
8. internal/rag            — Orquestrador
9. cmd/server              — API HTTP
10. cmd/chat               — CLI terminal
```

## Testes

- Testes unitários para cada componente usando interfaces (mocks)
- Testes de integração para o pipeline completo
- Use `_test.go` no mesmo pacote
- Table-driven tests quando aplicável

## Dependências Go Esperadas

```
go.mod:
  - modernc.org/sqlite         (SQLite pure Go, sem CGO)
  - github.com/google/uuid     (geração de UUIDs)
  - github.com/qdrant/go-client (client Qdrant)
  - github.com/joho/godotenv   (carregar .env)
  - github.com/ledongthuc/pdf  (parsing de PDF)
```
