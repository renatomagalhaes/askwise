# Plano de Implementação — AskWise

> Documento vivo que acompanha o progresso do desenvolvimento.
> Cada etapa é um entregável funcional que pode ser testado isoladamente.

## Visão Geral das Etapas

```
Etapa 0        Etapa 1         Etapa 2           Etapa 3
Foundation ──▶ Data Layer ──▶ Doc Processing ──▶ AI Integration
                                                       │
Etapa 6        Etapa 5         Etapa 4                 │
CLI Chat ◀── API Server ◀── RAG Pipeline ◀─────────────┘
```

| Etapa | Nome              | Entregável                                    | Status      |
|-------|-------------------|-----------------------------------------------|-------------|
| 0     | Foundation        | Projeto compila e roda via Docker              | `done`      |
| 1     | Data Layer        | SQLite + Qdrant funcionais com testes          | `done`      |
| 2     | Doc Processing    | Upload → Parse → Chunk funcional com testes    | `done`      |
| 3     | AI Integration    | Embeddings + LLM via OpenAI com testes         | `done`      |
| 4     | RAG Pipeline      | Pipeline completo: ingest + query              | `done`      |
| 5     | API Server        | API REST funcional com todos os endpoints      | `done`      |
| 6     | CLI Chat          | Chat no terminal, end-to-end funcional         | `done`      |

---

## Etapa 0: Foundation

**Objetivo**: Toda a infraestrutura de build, containers e tooling pronta. O projeto
compila dentro do Docker, roda testes, e o Makefile expõe todos os comandos necessários.
Nenhum Go precisa estar instalado localmente.

**Spec dirigindo**: RNF-03 (simplicidade), RNF-04 (observabilidade), RNF-06 (portabilidade)

### Tarefas

- [x] **0.1** — `Dockerfile` multi-stage (build + runtime)
- [x] **0.2** — `docker-compose.yml` com services: `app`, `chat`, `qdrant`
- [x] **0.3** — `Makefile` com targets: build, test, run, lint, chat, logs, clean
- [x] **0.4** — `go.mod` + `go.sum` inicializados (dentro do container)
- [x] **0.5** — `internal/config/` — Carregamento de variáveis de ambiente
- [x] **0.6** — `internal/logger/` — Logger estruturado JSON (STDOUT/STDERR)
- [x] **0.7** — Health check básico (`GET /api/v1/health` retorna 200)
- [x] **0.8** — ADR-001: Docker-first (sem Go local)
- [x] **0.9** — ADR-002: Logs estruturados JSON
- [x] **0.10** — OpenAPI spec inicial (`api/openapi.yaml`)

### Critério de Aceite

```bash
make build    # compila dentro do Docker sem erro
make test     # roda testes (mesmo sem testes reais ainda, pipeline funciona)
make up       # sobe qdrant + app
make health   # retorna {"status":"healthy"} em JSON
make down     # derruba tudo
make logs     # mostra logs JSON estruturados
```

### Entregável

Projeto vazio que compila, sobe, responde health check com logs JSON, e desce.
O desenvolvedor (ou IA) pode rodar `make up` e começar a trabalhar.

---

## Etapa 1: Data Layer ✅ `done`

**Objetivo**: Camada de dados funcional. SQLite armazena metadados de documentos.
Qdrant client cria collection e faz CRUD de vetores. Ambos com testes unitários.

**Spec dirigindo**: RF-05, RF-08, RF-09, RN-08, RN-21, design/02-MODELO-DADOS.md

### Tarefas

- [x] **1.1** — `internal/storage/` — Interface Storage + implementação SQLite
  - SaveDocument, GetDocument, ListDocuments, DeleteDocument, GetDocumentByName
  - Auto-create table + indexes na inicialização
- [x] **1.2** — `internal/storage/` — Testes unitários (table-driven)
- [x] **1.3** — `internal/vectorstore/` — Interface VectorStore + implementação Qdrant
  - EnsureCollection, Upsert, Search, DeleteByDocID (REST API via net/http)
- [x] **1.4** — `internal/vectorstore/` — Testes de integração (contra Qdrant real via Docker)
- [x] **1.5** — ADR-003: SQLite para metadados (decisão e alternativas)
- [x] **1.6** — ADR-004: Qdrant como vector store (decisão e alternativas)

### Critério de Aceite

```bash
make test                   # storage e vectorstore passam
make test-integration       # vectorstore testa contra Qdrant real
```

### Entregável

Os dois datastores funcionam isoladamente. Pode-se salvar/listar/deletar documentos
no SQLite e fazer upsert/search/delete de vetores no Qdrant.

---

## Etapa 2: Document Processing ✅ `done`

**Objetivo**: Receber um arquivo, extrair texto e dividir em chunks com metadados.
Cada formato suportado tem seu parser. Chunker implementa recursive splitting.

**Spec dirigindo**: RF-01, RF-02, RF-03, RN-01 a RN-09, design/01-ARQUITETURA.md §2.3-2.4

### Tarefas

- [x] **2.1** — `internal/document/` — Interface Parser + TextParser (TXT, MD)
- [x] **2.2** — `internal/document/` — CSVParser
- [x] **2.3** — `internal/document/` — StructuredParser (YAML, JSON)
- [x] **2.4** — `internal/document/` — PDFParser
- [x] **2.5** — `internal/document/` — Registry (detecta formato, retorna parser correto)
- [x] **2.6** — `internal/document/` — Testes unitários para cada parser
- [x] **2.7** — `internal/chunker/` — Interface Chunker + RecursiveChunker
  - chunk_size=500, overlap=50, separators=["\n\n", "\n", ". ", " "]
- [x] **2.8** — `internal/chunker/` — Testes unitários (tamanho, overlap, edge cases)

### Critério de Aceite

```bash
make test     # todos os parsers e chunker passam
# Deve passar: arquivo TXT de 10K palavras → ~20 chunks de ~500 tokens cada, com overlap
```

### Entregável

Dado um arquivo em qualquer formato suportado, o sistema extrai texto e divide em
chunks corretamente. Sem dependência de OpenAI ou Qdrant nesta etapa.

---

## Etapa 3: AI Integration ✅ `done`

**Objetivo**: Integração com OpenAI para gerar embeddings e chat completion.
Ambos com interfaces para permitir mock nos testes.

**Spec dirigindo**: RF-04, RF-07, RN-13 a RN-17, design/01-ARQUITETURA.md §2.5, §2.8

### Tarefas

- [x] **3.1** — `internal/embedding/` — Interface Embedder + implementação OpenAI
  - Embed(texts) e EmbedQuery(query)
  - Batch de até 100 textos por chamada
- [x] **3.2** — `internal/embedding/` — Testes unitários (com mock HTTP)
- [x] **3.3** — `internal/llm/` — Interface LLM + implementação OpenAI
  - ChatCompletion(messages) retorna string
  - System prompt configurável
- [x] **3.4** — `internal/llm/` — Testes unitários (com mock HTTP)
- [x] **3.5** — ADR-005: OpenAI como provider de IA (decisão e alternativas)

### Critério de Aceite

```bash
make test     # embedding e llm passam (mocks)
# Teste manual (requer OPENAI_API_KEY):
# Embed("hello world") retorna vetor de 1536 floats
# ChatCompletion("Olá") retorna resposta textual
```

### Entregável

Os dois clients OpenAI funcionam isoladamente. Pode-se gerar embeddings de qualquer
texto e fazer chat completion com mensagens arbitrárias.

---

## Etapa 4: RAG Pipeline ✅ `done`

**Objetivo**: Conectar todos os componentes no orquestrador RAG. Pipeline de ingestão
(upload → parse → chunk → embed → store) e pipeline de consulta (query → embed → search → generate).

**Spec dirigindo**: RF-07, RN-10 a RN-17, design/01-ARQUITETURA.md §2.7, §2.9, §3

### Tarefas

- [x] **4.1** — `internal/retriever/` — Interface Retriever + implementação
  - Combina Embedder + VectorStore
  - Aplica score threshold (RN-11) e diversidade de fontes (RN-12)
- [x] **4.2** — `internal/retriever/` — Testes unitários (com mocks)
- [x] **4.3** — `internal/rag/` — Orquestrador: método Ingest(file)
  - Parse → Chunk → Embed → Upsert + SaveDocument
- [x] **4.4** — `internal/rag/` — Orquestrador: método Query(question, history)
  - Embed → Search → Build prompt → ChatCompletion
  - System prompt com RN-13, RN-14, RN-15, RN-16
- [x] **4.5** — `internal/rag/` — Testes unitários (com mocks de todos os componentes)
- [x] **4.6** — Teste de integração do pipeline completo (requer Docker + OpenAI key)

### Critério de Aceite

```bash
make test               # retriever e rag passam (mocks)
make test-integration   # pipeline inteiro funciona: upload arquivo → pergunta → resposta
```

### Entregável

O pipeline RAG funciona end-to-end via código. Falta apenas expor via API e CLI.

---

## Etapa 5: API Server

**Objetivo**: API REST funcional com todos os endpoints definidos na spec.
Middleware de logging, recovery, validação. OpenAPI spec completa.

**Spec dirigindo**: RF-01, RF-08, RF-09, RN-01 a RN-05, RN-21, RN-22,
design/03-API-DESIGN.md

### Tarefas

- [x] **5.1** — `cmd/server/` — Setup do servidor HTTP (net/http)
- [x] **5.2** — `cmd/server/` — Middleware: Logger JSON, Recovery, MaxFileSize
- [x] **5.3** — `cmd/server/` — `POST /api/v1/documents` (upload + ingest pipeline)
  - Validação: formato (RN-01), tamanho (RN-02), conteúdo (RN-03)
  - Duplicatas (RN-04), sanitização (RN-05)
- [x] **5.4** — `cmd/server/` — `GET /api/v1/documents` (listagem)
- [x] **5.5** — `cmd/server/` — `GET /api/v1/documents/:id` (detalhes)
- [x] **5.6** — `cmd/server/` — `DELETE /api/v1/documents/:id` (remoção + RN-21)
- [x] **5.7** — `cmd/server/` — `GET /api/v1/health` (completo, verifica dependências)
- [x] **5.8** — OpenAPI spec finalizada (`api/openapi.yaml`)
- [x] **5.9** — Testes de integração da API (HTTP test)

### Critério de Aceite

```bash
make up                                                # sobe tudo
curl -X POST http://localhost:8484/api/v1/documents \
  -F "file=@testdata/sample.txt"                       # upload funciona
curl http://localhost:8484/api/v1/documents | jq       # lista documentos
curl http://localhost:8484/api/v1/health | jq          # health detalhado
make test                                              # testes HTTP passam
```

### Entregável

API REST totalmente funcional. Documentos podem ser enviados, listados e removidos.
O pipeline de ingestão roda automaticamente no upload.

---

## Etapa 6: CLI Chat

**Objetivo**: Chat interativo no terminal. O usuário digita perguntas e recebe
respostas baseadas nos documentos enviados, com fontes citadas.

**Spec dirigindo**: RF-06, RN-17 a RN-20, design/01-ARQUITETURA.md §2.2

### Tarefas

- [x] **6.1** — `cmd/chat/` — Loop principal de leitura (bufio.Scanner)
- [x] **6.2** — `cmd/chat/` — Comandos especiais: /help, /clear, /quit, /exit, /sources, /stats
- [x] **6.3** — `cmd/chat/` — Integração com RAG.Query()
- [x] **6.4** — `cmd/chat/` — Exibição formatada: resposta + fontes (RN-14)
- [x] **6.5** — `cmd/chat/` — Histórico da conversa em memória (RN-18, últimas 10 msgs)
- [x] **6.6** — `cmd/chat/` — Feedback visual: spinners/mensagens de progresso (RN-20)
- [x] **6.7** — Teste end-to-end: upload documento via API → chat pergunta → resposta correta
- [x] **6.8** — Documentos de exemplo para teste (TechSupport Ltda.)

### Critério de Aceite

```bash
make up                    # sobe infra + API
make upload FILE=sample.md # envia documento
make chat                  # abre chat interativo
# > "Como resolver o erro 5032?"
# < resposta baseada no documento, com fontes citadas
# > /sources
# < lista documentos indexados
# > /quit
```

### Entregável

Sistema completo e funcional. Upload via API, chat via terminal, respostas com fontes.

---

## Convenções Transversais (Todas as Etapas)

### Logs Estruturados (JSON)
Todos os logs vão para STDOUT (info) e STDERR (error) em formato JSON:
```json
{"level":"info","ts":"2025-01-15T10:30:00Z","msg":"document uploaded","doc_id":"abc","chunks":42}
{"level":"error","ts":"2025-01-15T10:30:05Z","msg":"failed to parse","file":"doc.pdf","error":"invalid format"}
```

### Testes
- Cada componente tem `_test.go` no mesmo pacote
- Table-driven tests como padrão
- Mocks via interfaces (sem framework de mock externo)
- `make test` roda todos os unitários
- `make test-integration` roda integração (precisa de Docker)

### ADRs
Architecture Decision Records em `docs/adr/` para toda decisão técnica relevante.
Formato: `NNN-titulo.md` com contexto, decisão, consequências.

### Docker-First
- `make build` compila via Docker
- `make test` roda testes via Docker
- `make chat` abre CLI via Docker
- Nenhum comando exige Go instalado localmente

### Commits
Um commit por tarefa concluída, mensagem referenciando o item do plano:
```
feat(storage): implement SQLite storage [1.1]
test(storage): add table-driven unit tests [1.2]
feat(vectorstore): implement Qdrant client [1.3]
```
