# Arquitetura do Sistema — AskWise

## 1. Visão Geral da Arquitetura

O AskWise segue uma arquitetura modular organizada em camadas, onde cada pacote Go tem
uma responsabilidade única e bem definida. O sistema possui dois pontos de entrada
(API HTTP e CLI) que compartilham a mesma lógica de negócio.

```
┌─────────────────────────────────────────────────────────────────────┐
│                        PONTOS DE ENTRADA                            │
│                                                                     │
│   ┌─────────────────────┐         ┌─────────────────────┐          │
│   │   cmd/server        │         │   cmd/chat           │          │
│   │   (API HTTP)        │         │   (CLI Terminal)     │          │
│   │                     │         │                      │          │
│   │  POST /documents    │         │  > pergunta          │          │
│   │  GET  /documents    │         │  < resposta          │          │
│   │  DEL  /documents/:id│         │                      │          │
│   └────────┬────────────┘         └──────────┬───────────┘          │
│            │                                  │                      │
└────────────┼──────────────────────────────────┼──────────────────────┘
             │                                  │
             ▼                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        CAMADA DE NEGÓCIO                            │
│                                                                     │
│   ┌─────────────────────────────────────────────────────────┐      │
│   │                    internal/rag                          │      │
│   │              (Orquestrador do Pipeline)                  │      │
│   │                                                         │      │
│   │  Ingest(file) ──▶ Parse ──▶ Chunk ──▶ Embed ──▶ Store  │      │
│   │  Query(question) ──▶ Embed ──▶ Search ──▶ Generate      │      │
│   └──────────┬──────────────────────────────────┬───────────┘      │
│              │                                  │                    │
│   ┌──────────▼──────────┐    ┌──────────────────▼───────────┐      │
│   │  internal/document  │    │  internal/retriever           │      │
│   │  (Parser de Docs)   │    │  (Busca de Contexto)          │      │
│   └──────────┬──────────┘    └──────────────────┬───────────┘      │
│              │                                  │                    │
│   ┌──────────▼──────────┐    ┌──────────────────▼───────────┐      │
│   │  internal/chunker   │    │  internal/llm                 │      │
│   │  (Divisor de Texto) │    │  (Chat Completion)            │      │
│   └──────────┬──────────┘    └──────────────────────────────┘      │
│              │                                                      │
└──────────────┼──────────────────────────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     CAMADA DE INFRAESTRUTURA                        │
│                                                                     │
│   ┌─────────────────────┐    ┌──────────────────────────────┐      │
│   │  internal/embedding │    │  internal/vectorstore         │      │
│   │  (OpenAI Embeddings)│    │  (Qdrant Client)             │      │
│   └──────────┬──────────┘    └──────────────┬───────────────┘      │
│              │                              │                       │
│              ▼                              ▼                       │
│   ┌─────────────────────┐    ┌──────────────────────────────┐      │
│   │   OpenAI API        │    │   Qdrant (Docker)            │      │
│   │   (Externo)         │    │   porta 6333/6334            │      │
│   └─────────────────────┘    └──────────────────────────────┘      │
│                                                                     │
│   ┌─────────────────────────────────────────────────────────┐      │
│   │  internal/storage                                        │      │
│   │  (SQLite - Metadados de Documentos)                     │      │
│   └─────────────────────────────────────────────────────────┘      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## 2. Componentes Principais

### 2.1 cmd/server — API HTTP

Responsável por expor endpoints REST para gerenciamento de documentos.

- **Framework**: `net/http` (standard library)
- **Router**: Implementação simples com `http.ServeMux`
- **Porta padrão**: 8484
- **Responsabilidades**:
  - Receber upload de arquivos (multipart/form-data)
  - Validar formato e tamanho
  - Acionar o pipeline de ingestão
  - Listar e deletar documentos

### 2.2 cmd/chat — CLI Interativo

Interface de terminal para conversar com a base de conhecimento.

- **I/O**: `bufio.Scanner` para leitura, `fmt` para escrita
- **Responsabilidades**:
  - Ler perguntas do usuário
  - Acionar o pipeline de consulta RAG
  - Exibir respostas formatadas com fontes
  - Gerenciar histórico da conversa
  - Processar comandos especiais (/help, /quit, etc.)

### 2.3 internal/document — Parser de Documentos

Extrai texto de diferentes formatos de arquivo.

```go
type Parser interface {
    Parse(reader io.Reader, filename string) (*Document, error)
    SupportedExtensions() []string
}
```

- **PDFParser**: Usa biblioteca Go para extrair texto de PDFs
- **CSVParser**: Converte linhas CSV em texto estruturado
- **TextParser**: Lê conteúdo direto (TXT, MD)
- **StructuredParser**: Converte YAML/JSON em texto legível

### 2.4 internal/chunker — Divisor de Texto

Divide documentos em pedaços menores para indexação.

```go
type Chunker interface {
    Chunk(doc *Document) ([]Chunk, error)
}
```

- **Estratégia**: Recursive character text splitting
- **Parâmetros**: chunk_size=500 tokens, overlap=50 tokens
- **Separadores**: `\n\n`, `\n`, `. `, ` `

### 2.5 internal/embedding — Geração de Embeddings

Converte texto em vetores numéricos usando a API da OpenAI.

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedQuery(ctx context.Context, query string) ([]float32, error)
}
```

- **Modelo**: `text-embedding-3-small`
- **Dimensões**: 1536
- **Batch**: Até 100 textos por chamada

### 2.6 internal/vectorstore — Armazenamento Vetorial

Interface com o Qdrant para armazenar e buscar vetores.

```go
type VectorStore interface {
    CreateCollection(ctx context.Context, name string, dimension int) error
    Upsert(ctx context.Context, collection string, points []Point) error
    Search(ctx context.Context, collection string, vector []float32, topK int) ([]SearchResult, error)
    DeleteByDocID(ctx context.Context, collection string, docID string) error
}
```

### 2.7 internal/retriever — Recuperação de Contexto

Combina embedding da query com busca vetorial para encontrar contexto relevante.

```go
type Retriever interface {
    Retrieve(ctx context.Context, query string, topK int) ([]RetrievedChunk, error)
}
```

### 2.8 internal/llm — Modelo de Linguagem

Interface com a API de chat completion da OpenAI.

```go
type LLM interface {
    ChatCompletion(ctx context.Context, messages []Message) (string, error)
}
```

### 2.9 internal/rag — Orquestrador

Coordena todo o pipeline, conectando os componentes.

```go
type RAG struct {
    parser     document.Parser
    chunker    chunker.Chunker
    embedder   embedding.Embedder
    store      vectorstore.VectorStore
    retriever  retriever.Retriever
    llm        llm.LLM
    storage    storage.Storage
}
```

### 2.10 internal/storage — Metadados (SQLite)

Armazena metadados dos documentos (não os embeddings).

```go
type Storage interface {
    SaveDocument(ctx context.Context, doc *DocumentMeta) error
    GetDocument(ctx context.Context, id string) (*DocumentMeta, error)
    ListDocuments(ctx context.Context) ([]DocumentMeta, error)
    DeleteDocument(ctx context.Context, id string) error
    GetDocumentByName(ctx context.Context, name string) (*DocumentMeta, error)
}
```

## 3. Fluxo de Dados

### 3.1 Pipeline de Ingestão (Upload)

```
Arquivo → Parser → Texto → Chunker → []Chunk → Embedder → []Vector → VectorStore
                                        │
                                        └──→ Storage (metadados SQLite)
```

1. Usuário envia arquivo via POST
2. Parser extrai texto do arquivo
3. Chunker divide texto em chunks de ~500 tokens
4. Embedder gera vetor de 1536 dimensões para cada chunk
5. VectorStore salva vetores + metadados no Qdrant
6. Storage salva metadados do documento no SQLite

### 3.2 Pipeline de Consulta (Chat)

```
Pergunta → Embedder → Vector → VectorStore.Search → []Chunks → LLM.Generate → Resposta
                                                        │
                                                        └── contexto para o prompt
```

1. Usuário digita pergunta no CLI
2. Embedder gera vetor da pergunta
3. VectorStore busca top-5 chunks mais similares
4. Retriever filtra chunks com score >= 0.5
5. LLM recebe prompt com (system + contexto + histórico + pergunta)
6. CLI exibe resposta + fontes citadas

## 4. Decisões de Arquitetura

### Por que Qdrant e não outro Vector Store?

| Opção           | Prós                                    | Contras                           |
|-----------------|-----------------------------------------|-----------------------------------|
| **Qdrant** ✅   | REST API, client Go, Docker fácil, grátis | Serviço externo (Docker)         |
| Pgvector        | Integrado ao PostgreSQL                 | Precisa de PostgreSQL rodando     |
| ChromeDB        | In-memory, simples                      | Sem client Go oficial             |
| Weaviate        | Rico em features                        | Complexo demais para PoC         |
| Pinecone        | Managed, escalável                      | Pago, vendor lock-in             |

### Por que SQLite para metadados?

- Zero configuração (é um arquivo local)
- Perfeito para PoC — sem necessidade de servidor de banco
- Go tem excelente suporte via `modernc.org/sqlite` (pure Go, sem CGO)
- Fácil de inspecionar dados durante desenvolvimento

### Por que interfaces em todos os componentes?

- **Testabilidade**: Cada componente pode ser mockado em testes
- **Flexibilidade**: Trocar OpenAI por outro provider sem mudar o código
- **Didático**: Mostra boas práticas de design em Go

## 5. Observabilidade — Logs Estruturados JSON (ADR-002)

Todos os logs são emitidos em formato JSON estruturado via `log/slog` (standard library).

### Destino

| Nível        | Destino | Descrição                                        |
|--------------|---------|--------------------------------------------------|
| DEBUG, INFO  | STDOUT  | Eventos normais do sistema                       |
| WARN, ERROR  | STDERR  | Situações anômalas ou falhas                     |

### Formato

```json
{"time":"2025-01-15T10:30:00Z","level":"INFO","msg":"document uploaded","component":"server","doc_id":"abc","chunks":42,"duration_ms":1523}
{"time":"2025-01-15T10:30:05Z","level":"ERROR","msg":"failed to parse","component":"document","file":"doc.pdf","error":"invalid format"}
```

### Campos Padrão

Todos os logs devem incluir pelo menos `component` para identificar a origem:

- `component`: server, chat, rag, storage, vectorstore, embedding, llm
- Campos contextuais variam: `doc_id`, `file_name`, `chunks`, `duration_ms`, `error`

### Implementação

```go
// internal/logger/ usa slog.JSONHandler direcionado para os.Stdout
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
logger.Info("document uploaded", "component", "server", "doc_id", docID, "chunks", 42)
```

## 6. Configuração

Toda configuração via variáveis de ambiente (arquivo `.env`):

```env
# OpenAI
OPENAI_API_KEY=sk-...
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
OPENAI_CHAT_MODEL=gpt-4o-mini

# Qdrant
QDRANT_HOST=localhost
QDRANT_PORT=6333
QDRANT_COLLECTION=askwise

# Server
SERVER_PORT=8484
MAX_FILE_SIZE=10485760  # 10MB

# RAG
RAG_CHUNK_SIZE=500
RAG_CHUNK_OVERLAP=50
RAG_TOP_K=5
RAG_SCORE_THRESHOLD=0.5

# Storage
SQLITE_PATH=./data/askwise.db
```

## 7. Infraestrutura — Docker-First (ADR-001)

Todo o build, teste e execução acontecem dentro de containers Docker.
Nenhuma instalação local de Go é necessária.

### Dockerfile Multi-Stage

```
┌──────────────────────────────────────────────┐
│  Stage: builder                               │
│  golang:1.26.1-alpine                          │
│  Compila: askwise-server + askwise-chat       │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│  Stage: runtime                               │
│  alpine:3.20 (imagem mínima)                  │
│  Apenas binários compilados                   │
│  Usuário não-root                             │
└──────────────────────────────────────────────┘

┌──────────────────────────────────────────────┐
│  Stage: dev                                   │
│  golang:1.26.1-alpine + make + curl + jq       │
│  Volume mount do código fonte                 │
│  Para desenvolvimento com hot reload          │
└──────────────────────────────────────────────┘
```

### Docker Compose

```
┌────────────────┐  ┌────────────────┐  ┌────────────────┐
│    app          │  │    chat         │  │    qdrant       │
│  (API HTTP)     │  │  (CLI Terminal) │  │  (Vector Store) │
│  :8484          │  │  stdin/tty      │  │  :6333/:6334    │
│                 │  │                 │  │                 │
│  stage: dev     │  │  stage: dev     │  │  qdrant:v1.13   │
│  volume: .:/app │  │  volume: .:/app │  │  volume: data   │
└────────┬────────┘  └────────┬────────┘  └────────────────┘
         │                    │                    ▲
         └────────────────────┴────────────────────┘
                    depends_on: qdrant
```

### Makefile

Todos os comandos disponíveis via `make help`:

| Comando              | Descrição                                    |
|----------------------|----------------------------------------------|
| `make up`            | Sobe toda a infra (qdrant + app)             |
| `make down`          | Derruba todos os containers                  |
| `make build`         | Builda as imagens Docker                     |
| `make test`          | Roda testes unitários (dentro do container)  |
| `make test-integration` | Roda testes de integração                 |
| `make chat`          | Abre o chat interativo                       |
| `make logs`          | Mostra logs JSON de todos os serviços        |
| `make health`        | Health check da API                          |
| `make upload FILE=x` | Upload de documento                         |
| `make dev-shell`     | Shell dentro do container de desenvolvimento |
| `make clean`         | Remove containers, volumes e dados           |

## 8. Especificação da API — OpenAPI

A API REST está especificada em `api/openapi.yaml` seguindo OpenAPI 3.1.0.
O arquivo pode ser visualizado em ferramentas como Swagger UI, Redoc ou qualquer
editor com suporte a OpenAPI.

Endpoints documentados:
- `GET  /api/v1/health` — Health check
- `POST /api/v1/documents` — Upload de documento
- `GET  /api/v1/documents` — Listar documentos
- `GET  /api/v1/documents/:id` — Detalhes de um documento
- `DELETE /api/v1/documents/:id` — Remover documento
