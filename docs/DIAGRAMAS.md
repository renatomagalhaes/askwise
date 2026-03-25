# Diagramas — AskWise

Diagramas de arquitetura e fluxo do sistema AskWise, usando notação
[Mermaid](https://mermaid.js.org/) para renderização nativa no GitHub.

---

## 1. C4 — Contexto do Sistema

Visão de mais alto nível: quem interage com o AskWise e quais sistemas externos ele utiliza.

```mermaid
C4Context
    title AskWise — Diagrama de Contexto

    Person(atendente, "Atendente de Suporte", "Faz perguntas sobre a base de conhecimento via terminal")
    Person(admin, "Administrador", "Envia e gerencia documentos via API REST")

    System(askwise, "AskWise", "Chatbot RAG que responde perguntas com base em documentos enviados")

    System_Ext(openai, "OpenAI API", "Gera embeddings vetoriais e respostas via LLM")
    System_Ext(qdrant, "Qdrant", "Banco de dados vetorial para busca por similaridade")

    Rel(admin, askwise, "Upload/listagem/remoção de documentos", "HTTP REST")
    Rel(atendente, askwise, "Perguntas e respostas", "CLI Terminal")
    Rel(askwise, openai, "Embeddings + Chat Completion", "HTTPS")
    Rel(askwise, qdrant, "Armazena e busca vetores", "HTTP REST")
```

---

## 2. C4 — Containers

Os containers que compõem o sistema e como se comunicam.

```mermaid
C4Container
    title AskWise — Diagrama de Containers

    Person(user, "Usuário")

    Container_Boundary(askwise, "AskWise") {
        Container(api, "API Server", "Go / net/http", "Recebe uploads, lista e remove documentos. Aciona pipeline de ingestão.")
        Container(cli, "CLI Chat", "Go / bufio", "Chat interativo no terminal. Envia perguntas ao pipeline RAG.")
        ContainerDb(sqlite, "SQLite", "modernc.org/sqlite", "Metadados dos documentos: id, nome, tipo, status, chunk_count")
    }

    System_Ext(qdrant, "Qdrant", "Vector store: armazena embeddings e busca por similaridade cosine")
    System_Ext(openai, "OpenAI API", "text-embedding-3-small + gpt-4o-mini")

    Rel(user, api, "curl / HTTP", "POST, GET, DELETE")
    Rel(user, cli, "Terminal", "stdin/stdout")
    Rel(api, sqlite, "Lê/escreve metadados", "SQL")
    Rel(cli, sqlite, "Lê metadados", "SQL")
    Rel(api, qdrant, "Upsert/Delete vetores", "HTTP REST")
    Rel(cli, qdrant, "Search vetores", "HTTP REST")
    Rel(api, openai, "Gera embeddings", "HTTPS")
    Rel(cli, openai, "Embeddings + Chat Completion", "HTTPS")
```

---

## 3. C4 — Componentes (Pacotes Go)

Os pacotes internos do AskWise e suas dependências.

```mermaid
graph TB
    subgraph "Pontos de Entrada"
        SERVER["cmd/server<br/><i>API HTTP :8484</i>"]
        CHAT["cmd/chat<br/><i>CLI Terminal</i>"]
    end

    subgraph "Orquestração"
        RAG["internal/rag<br/><i>Pipeline RAG</i><br/>Ingest · Query · Delete"]
    end

    subgraph "Processamento de Documentos"
        DOC["internal/document<br/><i>Registry + Parsers</i><br/>PDF · CSV · TXT · JSON · YAML · MD"]
        CHUNK["internal/chunker<br/><i>RecursiveChunker</i><br/>Divide texto em chunks"]
    end

    subgraph "Inteligência Artificial"
        EMBED["internal/embedding<br/><i>OpenAI Embedder</i><br/>text-embedding-3-small"]
        LLM["internal/llm<br/><i>OpenAI LLM</i><br/>gpt-4o-mini"]
        RET["internal/retriever<br/><i>SemanticRetriever</i><br/>Busca + filtragem + diversidade"]
    end

    subgraph "Infraestrutura"
        VS["internal/vectorstore<br/><i>Qdrant Client</i><br/>REST API"]
        STORE["internal/storage<br/><i>SQLite</i><br/>Metadados de documentos"]
        CFG["internal/config<br/><i>Variáveis de ambiente</i>"]
        LOG["internal/logger<br/><i>slog JSON</i>"]
    end

    subgraph "Externos"
        OPENAI["OpenAI API"]
        QDRANT["Qdrant DB"]
        SQLFILE["SQLite File"]
    end

    SERVER --> RAG
    CHAT --> RAG
    SERVER --> STORE
    CHAT --> STORE

    RAG --> DOC
    RAG --> CHUNK
    RAG --> EMBED
    RAG --> VS
    RAG --> RET
    RAG --> LLM
    RAG --> STORE

    RET --> EMBED
    RET --> VS

    EMBED --> OPENAI
    LLM --> OPENAI
    VS --> QDRANT
    STORE --> SQLFILE

    SERVER --> CFG
    CHAT --> CFG
    SERVER --> LOG
    CHAT --> LOG

    style SERVER fill:#4CAF50,color:#fff
    style CHAT fill:#4CAF50,color:#fff
    style RAG fill:#FF9800,color:#fff
    style DOC fill:#2196F3,color:#fff
    style CHUNK fill:#2196F3,color:#fff
    style EMBED fill:#9C27B0,color:#fff
    style LLM fill:#9C27B0,color:#fff
    style RET fill:#9C27B0,color:#fff
    style VS fill:#607D8B,color:#fff
    style STORE fill:#607D8B,color:#fff
    style CFG fill:#607D8B,color:#fff
    style LOG fill:#607D8B,color:#fff
    style OPENAI fill:#333,color:#fff
    style QDRANT fill:#333,color:#fff
    style SQLFILE fill:#333,color:#fff
```

---

## 4. Sequência — Pipeline de Ingestão (Upload)

O fluxo completo quando um documento é enviado via `POST /api/v1/documents`.

```mermaid
sequenceDiagram
    actor User as Usuário
    participant API as API Server
    participant RAG as RAG Orchestrator
    participant Store as SQLite
    participant Doc as Document Parser
    participant Chk as Chunker
    participant Emb as OpenAI Embeddings
    participant VS as Qdrant

    User->>API: POST /api/v1/documents (file)
    API->>API: Validar formato (RN-01)
    API->>API: Validar tamanho (RN-02)

    API->>RAG: Ingest(file, filename, size)

    RAG->>Store: GetDocumentByName(name)
    Store-->>RAG: existing doc (ou nil)

    opt Documento duplicado (RN-04)
        RAG->>VS: DeleteByDocID(old_id)
        RAG->>Store: DeleteDocument(old_id)
    end

    RAG->>Store: SaveDocument(status: "processing")

    RAG->>Doc: Parse(file, filename)
    Doc-->>RAG: Document{content, sections}

    RAG->>Chk: Chunk(document)
    Chk-->>RAG: []Chunk (com texto e metadados)

    RAG->>RAG: Gerar ChunkIDs (UUID v5)

    RAG->>Emb: Embed([]texts)
    Emb->>Emb: Batch (até 100 por chamada)
    Emb-->>RAG: [][]float32 (vetores 1536d)

    RAG->>VS: Upsert(collection, points)
    VS-->>RAG: OK

    RAG->>Store: SaveDocument(status: "ready", chunk_count)
    Store-->>RAG: OK

    RAG-->>API: IngestResult{doc_id, chunks}
    API-->>User: 201 Created {id, name, chunk_count, message}
```

---

## 5. Sequência — Pipeline de Consulta (Chat)

O fluxo quando o usuário faz uma pergunta no CLI.

```mermaid
sequenceDiagram
    actor User as Atendente
    participant CLI as CLI Chat
    participant RAG as RAG Orchestrator
    participant Ret as Semantic Retriever
    participant Emb as OpenAI Embeddings
    participant VS as Qdrant
    participant LLM as OpenAI LLM

    User->>CLI: "Como resolver o erro 5032?"
    CLI->>CLI: Mostrar spinner (RN-20)

    CLI->>RAG: Query(question, history)

    RAG->>Ret: Retrieve(question, topK=5)
    Ret->>Emb: EmbedQuery(question)
    Emb-->>Ret: []float32 (vetor 1536d)

    Ret->>VS: Search(collection, vector, topK=10)
    VS-->>Ret: []SearchResult{score, payload}

    Ret->>Ret: Filtrar score >= 0.5 (RN-11)
    Ret->>Ret: Aplicar diversidade de fontes (RN-12)
    Ret-->>RAG: []RetrievedChunk (top 5)

    alt Chunks encontrados
        RAG->>RAG: Montar system prompt + contexto (RN-13)
        RAG->>RAG: Incluir histórico (RN-18)

        RAG->>LLM: ChatCompletion(messages)
        LLM-->>RAG: resposta em texto

        RAG->>RAG: Extrair fontes (RN-14)
        RAG-->>CLI: QueryResponse{answer, sources}
    else Nenhum chunk relevante
        RAG-->>CLI: "Não encontrei informações..." (RN-17)
    end

    CLI->>CLI: Parar spinner
    CLI->>CLI: Exibir resposta formatada
    CLI->>CLI: Exibir fontes com relevância
    CLI->>CLI: Atualizar histórico (máx 10 msgs)
    CLI-->>User: Resposta + 📎 Fontes
```

---

## 6. Fluxo — Validação de Upload

Decisões tomadas durante o upload de um documento.

```mermaid
flowchart TD
    A[POST /api/v1/documents] --> B{Campo 'file' presente?}
    B -->|Não| B1[400 invalid_request]
    B -->|Sim| C{Formato suportado? RN-01}
    C -->|Não| C1["400 unsupported_format<br/>Listar formatos aceitos"]
    C -->|Sim| D{Tamanho <= 10MB? RN-02}
    D -->|Não| D1[413 file_too_large]
    D -->|Sim| E[RAG.Ingest]

    E --> F[Parser.Parse]
    F --> G{Conteúdo extraído? RN-03}
    G -->|Não| G1[422 empty_content]
    G -->|Sim| H{Duplicata? RN-04}

    H -->|Sim| I[Remover doc anterior<br/>Qdrant + SQLite]
    H -->|Não| J[Continuar]
    I --> J

    J --> K[Sanitizar nome RN-05]
    K --> L[Salvar status 'processing']
    L --> M[Chunker.Chunk]
    M --> N[Embedder.Embed]
    N --> O[VectorStore.Upsert]
    O --> P[Salvar status 'ready']
    P --> Q[201 Created]

    N -.->|Erro| R[Salvar status 'error']
    O -.->|Erro| R
    R --> S[500 ingest_error]

    style A fill:#4CAF50,color:#fff
    style Q fill:#4CAF50,color:#fff
    style B1 fill:#f44336,color:#fff
    style C1 fill:#f44336,color:#fff
    style D1 fill:#f44336,color:#fff
    style G1 fill:#f44336,color:#fff
    style S fill:#f44336,color:#fff
```

---

## 7. Fluxo — Loop do CLI Chat

Interação do usuário com o chat interativo.

```mermaid
flowchart TD
    A[Iniciar Chat] --> B[Carregar Config]
    B --> C[Inicializar Componentes<br/>SQLite · Qdrant · OpenAI · RAG]
    C --> D[Exibir Banner]
    D --> E[Aguardar Input]

    E --> F{Ler linha}
    F -->|EOF / Ctrl+D| Z[Sair]
    F -->|Texto vazio| E
    F -->|Tem texto| G{Começa com /?}

    G -->|Sim| H{Qual comando?}
    H -->|/quit /exit| Z
    H -->|/help| I[Mostrar comandos] --> E
    H -->|/clear| J[Limpar histórico] --> E
    H -->|/sources| K[Listar documentos<br/>via Storage] --> E
    H -->|/stats| L[Mostrar estatísticas] --> E
    H -->|outro| M["⚠ Comando desconhecido"] --> E

    G -->|Não| N[Mostrar spinner]
    N --> O[RAG.Query<br/>pergunta + histórico]
    O --> P{Sucesso?}
    P -->|Não| Q["❌ Exibir erro"] --> E
    P -->|Sim| R[Exibir resposta]
    R --> S[Exibir fontes 📎]
    S --> T[Atualizar histórico<br/>máx 10 mensagens]
    T --> E

    Z --> U["👋 Até mais!"]

    style A fill:#4CAF50,color:#fff
    style Z fill:#607D8B,color:#fff
    style U fill:#607D8B,color:#fff
    style Q fill:#f44336,color:#fff
```

---

## 8. Infraestrutura — Docker Compose

Como os containers se relacionam na stack Docker.

```mermaid
graph LR
    subgraph "Docker Compose"
        subgraph "app (API Server)"
            API["Go HTTP :8484<br/>stage: dev<br/>volume: .:/app"]
        end

        subgraph "chat (CLI)"
            CLI["Go CLI<br/>stage: dev<br/>stdin_open: true<br/>profile: chat"]
        end

        subgraph "qdrant (Vector DB)"
            QD["Qdrant v1.13<br/>:6333 REST<br/>:6334 gRPC<br/>volume: qdrant_data"]
        end
    end

    subgraph "Volumes"
        V1["qdrant_data"]
        V2["go_mod_cache"]
        V3["go_build_cache"]
    end

    subgraph "Externo"
        ENV[".env<br/>OPENAI_API_KEY<br/>configs"]
        OAI["OpenAI API<br/>api.openai.com"]
        DATA["data/askwise.db<br/>SQLite"]
    end

    API -->|depends_on| QD
    CLI -->|depends_on| QD
    API --> OAI
    CLI --> OAI
    API --> DATA
    CLI --> DATA
    API -.-> ENV
    CLI -.-> ENV
    QD --> V1
    API --> V2
    API --> V3

    style API fill:#4CAF50,color:#fff
    style CLI fill:#4CAF50,color:#fff
    style QD fill:#e91e63,color:#fff
```

---

## 9. Modelo de Dados

Relação entre as entidades armazenadas no SQLite e no Qdrant.

```mermaid
erDiagram
    DOCUMENT ||--o{ CHUNK : "1 documento tem N chunks"

    DOCUMENT {
        string id PK "UUID v4"
        string name UK "Nome sanitizado"
        string original_name "Nome original"
        string file_type "pdf, csv, txt, yaml, json, md"
        int file_size "Bytes"
        int chunk_count "Total de chunks"
        string status "processing | ready | error"
        string error_message "Opcional"
        datetime created_at "Auto"
        datetime updated_at "Auto"
    }

    CHUNK {
        string id PK "UUID v5 (doc_id + index)"
        string document_id FK "Ref ao documento"
        int chunk_index "Posição no documento"
        string text "Conteúdo textual"
        string file_name "Nome do arquivo"
        string file_type "Tipo do arquivo"
        float_array vector "1536 dimensões"
    }

    DOCUMENT }|--|| SQLITE : "Metadados"
    CHUNK }|--|| QDRANT : "Vetores + Payload"

    SQLITE {
        string storage "data/askwise.db"
        string tabela "documents"
    }

    QDRANT {
        string storage "Docker volume"
        string collection "askwise"
        string distance "Cosine"
    }
```

---

## Legenda de Cores

| Cor | Significado |
|-----|-------------|
| 🟢 Verde | Pontos de entrada (API, CLI) |
| 🟠 Laranja | Orquestração (RAG) |
| 🔵 Azul | Processamento de documentos |
| 🟣 Roxo | Inteligência artificial (OpenAI) |
| ⚫ Cinza | Infraestrutura e storage |
| 🔴 Vermelho | Erros e respostas de falha |
