# Modelo de Dados — AskWise

## 1. Visão Geral

O AskWise usa dois tipos de armazenamento com propósitos distintos:

```
┌────────────────────────────┐     ┌────────────────────────────────────┐
│       SQLite               │     │           Qdrant                    │
│    (Metadados)             │     │        (Vetores)                    │
│                            │     │                                     │
│  ┌──────────────────┐     │     │  Collection: "askwise"              │
│  │   documents       │     │     │  ┌───────────────────────────────┐ │
│  │                  │     │     │  │  Points (Chunks + Embeddings) │ │
│  │  id              │     │     │  │                               │ │
│  │  name            │     │     │  │  id (UUID)                    │ │
│  │  original_name   │     │     │  │  vector [1536 floats]         │ │
│  │  file_type       │     │     │  │  payload:                     │ │
│  │  file_size       │     │     │  │    document_id                │ │
│  │  chunk_count     │     │     │  │    chunk_index                │ │
│  │  status          │     │     │  │    text                       │ │
│  │  created_at      │     │     │  │    file_name                  │ │
│  │  updated_at      │     │     │  │    file_type                  │ │
│  └──────────────────┘     │     │  └───────────────────────────────┘ │
└────────────────────────────┘     └────────────────────────────────────┘
         │                                        │
         │         Relação lógica                  │
         │    (document_id vincula                 │
         └──────── chunks ao documento) ───────────┘
```

## 2. SQLite — Tabela `documents`

Armazena metadados sobre cada documento enviado. Não armazena o conteúdo nem os embeddings.

### Schema

```sql
CREATE TABLE IF NOT EXISTS documents (
    -- Identificador único do documento (UUID v4)
    id            TEXT PRIMARY KEY,

    -- Nome sanitizado do arquivo (ex: "manual-cloudapi-v3.pdf")
    name          TEXT NOT NULL UNIQUE,

    -- Nome original do arquivo como enviado pelo usuário
    original_name TEXT NOT NULL,

    -- Tipo do arquivo: "pdf", "csv", "txt", "yaml", "json", "md"
    file_type     TEXT NOT NULL,

    -- Tamanho do arquivo original em bytes
    file_size     INTEGER NOT NULL,

    -- Quantidade de chunks gerados a partir deste documento
    chunk_count   INTEGER NOT NULL DEFAULT 0,

    -- Status do processamento: "processing", "ready", "error"
    status        TEXT NOT NULL DEFAULT 'processing',

    -- Mensagem de erro, se houver
    error_message TEXT,

    -- Data/hora de criação (RFC3339)
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Data/hora da última atualização (RFC3339)
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Índice para busca por nome (usado na detecção de duplicatas)
CREATE INDEX IF NOT EXISTS idx_documents_name ON documents(name);

-- Índice para busca por status
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
```

### Exemplo de Registro

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "runbook-datasync.md",
  "original_name": "Runbook - DataSync (v2.1).md",
  "file_type": "md",
  "file_size": 45230,
  "chunk_count": 12,
  "status": "ready",
  "error_message": null,
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:05Z"
}
```

### Ciclo de Vida do Status

```
upload recebido ──▶ "processing" ──▶ "ready"
                         │
                         └──▶ "error" (se falhar)
```

## 3. Qdrant — Collection `askwise`

Armazena os embeddings vetoriais e o texto dos chunks para busca por similaridade.

### Configuração da Collection

```json
{
  "collection_name": "askwise",
  "vectors": {
    "size": 1536,
    "distance": "Cosine"
  }
}
```

| Campo       | Valor      | Motivo                                         |
|-------------|------------|------------------------------------------------|
| `size`      | 1536       | Dimensão do modelo text-embedding-3-small      |
| `distance`  | Cosine     | Métrica padrão para busca semântica de texto   |

### Estrutura de um Point (Chunk)

Cada chunk é armazenado como um "point" no Qdrant:

```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "vector": [0.0123, -0.0456, 0.0789, ...],
  "payload": {
    "document_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "chunk_index": 3,
    "text": "Para resetar o cache do CloudAPI, execute os seguintes passos:\n1. Acesse o painel de administração...",
    "file_name": "runbook-cloudapi.md",
    "file_type": "md"
  }
}
```

### Campos do Payload

| Campo          | Tipo     | Descrição                                        |
|----------------|----------|--------------------------------------------------|
| `document_id`  | string   | UUID do documento pai (FK lógica para SQLite)    |
| `chunk_index`  | integer  | Posição do chunk dentro do documento (0-based)   |
| `text`         | string   | Texto original do chunk (usado na resposta)      |
| `file_name`    | string   | Nome do arquivo fonte (para citação)             |
| `file_type`    | string   | Tipo do arquivo fonte                            |

## 4. Structs Go

### Document (Metadado)

```go
// DocumentMeta representa os metadados de um documento armazenado no SQLite.
type DocumentMeta struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    OriginalName string    `json:"original_name"`
    FileType     string    `json:"file_type"`
    FileSize     int64     `json:"file_size"`
    ChunkCount   int       `json:"chunk_count"`
    Status       string    `json:"status"`
    ErrorMessage string    `json:"error_message,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

### Document (Conteúdo Parseado)

```go
// Document representa o conteúdo extraído de um arquivo após parsing.
type Document struct {
    Name     string   // Nome do arquivo
    FileType string   // Tipo do arquivo (pdf, csv, txt, etc.)
    Content  string   // Texto completo extraído
    Sections []string // Seções identificadas (se aplicável)
}
```

### Chunk

```go
// Chunk representa um pedaço de texto de um documento, pronto para embedding.
type Chunk struct {
    ID         string // UUID único do chunk
    DocumentID string // UUID do documento pai
    Index      int    // Posição dentro do documento
    Text       string // Conteúdo textual do chunk
    FileName   string // Nome do arquivo fonte
    FileType   string // Tipo do arquivo fonte
}
```

### Point (Vetor no Qdrant)

```go
// Point representa um vetor armazenado no Qdrant com seus metadados.
type Point struct {
    ID      string             // UUID do point (mesmo do chunk)
    Vector  []float32          // Embedding de 1536 dimensões
    Payload map[string]any     // Metadados (document_id, text, etc.)
}
```

### SearchResult

```go
// SearchResult representa um resultado de busca vetorial.
type SearchResult struct {
    ID       string         // UUID do point/chunk
    Score    float32        // Score de similaridade (0.0 a 1.0)
    Payload  map[string]any // Metadados do chunk
}
```

### RetrievedChunk

```go
// RetrievedChunk representa um chunk recuperado e pronto para uso no prompt.
type RetrievedChunk struct {
    Text     string  // Texto do chunk
    FileName string  // Arquivo fonte
    FileType string  // Tipo do arquivo
    Score    float32 // Relevância (cosine similarity)
    Index    int     // Posição no documento original
}
```

## 5. Relacionamentos

```
documents (SQLite)          points (Qdrant)
┌────────────────┐         ┌────────────────────┐
│ id (PK)        │────┐    │ id (PK)            │
│ name           │    │    │ vector             │
│ file_type      │    │    │ payload:           │
│ chunk_count    │    └───▶│   document_id (FK) │
│ status         │         │   chunk_index      │
│ ...            │         │   text             │
└────────────────┘         │   file_name        │
                           │   file_type        │
     1 documento           └────────────────────┘
         │
         │ tem muitos
         ▼
     N chunks (points)
```

## 6. Estratégia de IDs

- **document_id**: UUID v4 gerado no momento do upload
- **chunk/point_id**: UUID v5 derivado de `document_id + chunk_index`
  - Garante determinismo: re-upload gera os mesmos IDs
  - Permite upsert sem duplicação

```go
// Gera ID determinístico para um chunk baseado no doc e índice
func ChunkID(documentID string, chunkIndex int) string {
    namespace := uuid.MustParse(documentID)
    name := fmt.Sprintf("chunk-%d", chunkIndex)
    return uuid.NewSHA1(namespace, []byte(name)).String()
}
```
