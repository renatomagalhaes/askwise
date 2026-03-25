# API Design — AskWise

## 1. Visão Geral

A API REST do AskWise é simples e focada em gerenciamento de documentos. Não existe
endpoint de chat — o chat acontece exclusivamente via CLI no terminal.

**Base URL**: `http://localhost:8080/api/v1`

## 2. Endpoints

### 2.1 Health Check

Verifica se o servidor está rodando e se as dependências estão acessíveis.

```
GET /api/v1/health
```

**Response 200:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "dependencies": {
    "qdrant": "connected",
    "sqlite": "connected",
    "openai": "configured"
  }
}
```

**Response 503:**
```json
{
  "status": "unhealthy",
  "version": "1.0.0",
  "dependencies": {
    "qdrant": "disconnected",
    "sqlite": "connected",
    "openai": "configured"
  }
}
```

### 2.2 Upload de Documento

Recebe um arquivo e inicia o pipeline de ingestão.

```
POST /api/v1/documents
Content-Type: multipart/form-data
```

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/documents \
  -F "file=@manual-cloudapi-v3.pdf"
```

**Response 201 (Created):**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "manual-cloudapi-v3.pdf",
  "file_type": "pdf",
  "file_size": 245760,
  "chunk_count": 42,
  "status": "ready",
  "created_at": "2025-01-15T10:30:00Z",
  "message": "Documento processado com sucesso. 42 chunks indexados."
}
```

**Response 400 (Bad Request) — Formato inválido:**
```json
{
  "error": "unsupported_format",
  "message": "Formato '.docx' não suportado. Formatos aceitos: pdf, csv, txt, yaml, json, md",
  "supported_formats": ["pdf", "csv", "txt", "yaml", "yml", "json", "md"]
}
```

**Response 413 (Payload Too Large):**
```json
{
  "error": "file_too_large",
  "message": "Arquivo excede o limite de 10MB. Tamanho recebido: 15.3MB"
}
```

**Response 422 (Unprocessable Entity) — Sem conteúdo:**
```json
{
  "error": "empty_content",
  "message": "Não foi possível extrair texto do arquivo. Verifique se o arquivo não está vazio ou corrompido."
}
```

### 2.3 Listar Documentos

Retorna todos os documentos indexados.

```
GET /api/v1/documents
```

**Response 200:**
```json
{
  "documents": [
    {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "name": "manual-cloudapi-v3.pdf",
      "file_type": "pdf",
      "file_size": 245760,
      "chunk_count": 42,
      "status": "ready",
      "created_at": "2025-01-15T10:30:00Z"
    },
    {
      "id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
      "name": "runbook-datasync.md",
      "file_type": "md",
      "file_size": 45230,
      "chunk_count": 12,
      "status": "ready",
      "created_at": "2025-01-15T11:00:00Z"
    }
  ],
  "total": 2,
  "total_chunks": 54
}
```

### 2.4 Detalhes de um Documento

Retorna informações detalhadas de um documento específico.

```
GET /api/v1/documents/:id
```

**Response 200:**
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "manual-cloudapi-v3.pdf",
  "original_name": "Manual - CloudAPI (v3.0).pdf",
  "file_type": "pdf",
  "file_size": 245760,
  "chunk_count": 42,
  "status": "ready",
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:05Z"
}
```

**Response 404:**
```json
{
  "error": "not_found",
  "message": "Documento com ID 'xyz' não encontrado"
}
```

### 2.5 Remover Documento

Remove um documento e todos os seus chunks do sistema.

```
DELETE /api/v1/documents/:id
```

**Response 200:**
```json
{
  "message": "Documento 'manual-cloudapi-v3.pdf' removido com sucesso. 42 chunks deletados."
}
```

**Response 404:**
```json
{
  "error": "not_found",
  "message": "Documento com ID 'xyz' não encontrado"
}
```

## 3. Padrões da API

### 3.1 Content-Type

| Operação | Request Content-Type     | Response Content-Type |
|----------|--------------------------|-----------------------|
| Upload   | `multipart/form-data`    | `application/json`    |
| Demais   | N/A                      | `application/json`    |

### 3.2 Códigos HTTP

| Código | Significado          | Quando usar                              |
|--------|----------------------|------------------------------------------|
| 200    | OK                   | GET, DELETE com sucesso                  |
| 201    | Created              | POST upload com sucesso                  |
| 400    | Bad Request          | Formato inválido, campo faltando         |
| 404    | Not Found            | Documento não existe                     |
| 413    | Payload Too Large    | Arquivo excede 10MB                      |
| 422    | Unprocessable Entity | Arquivo vazio ou ilegível                |
| 500    | Internal Server Error| Erro inesperado no servidor              |
| 503    | Service Unavailable  | Dependência indisponível (Qdrant, etc.)  |

### 3.3 Formato de Erro

Todos os erros seguem o mesmo formato:

```json
{
  "error": "error_code",
  "message": "Mensagem legível explicando o erro e possível solução"
}
```

### 3.4 CORS

Para esta PoC, CORS não é necessário (não há frontend web).

## 4. Middleware

A API utiliza middleware encadeado para funcionalidades transversais:

```
Request ──▶ Logger ──▶ Recovery ──▶ MaxFileSize ──▶ Handler ──▶ Response
```

- **Logger**: Registra método, path, status code e duração de cada request
- **Recovery**: Captura panics e retorna 500 com mensagem genérica
- **MaxFileSize**: Limita o tamanho do body (aplicado apenas no upload)

## 5. Exemplo de Uso Completo

```bash
# 1. Verificar se o servidor está saudável
curl http://localhost:8080/api/v1/health | jq

# 2. Upload de documentos
curl -X POST http://localhost:8080/api/v1/documents \
  -F "file=@docs/manual-cloudapi-v3.pdf" | jq

curl -X POST http://localhost:8080/api/v1/documents \
  -F "file=@docs/runbook-datasync.md" | jq

curl -X POST http://localhost:8080/api/v1/documents \
  -F "file=@docs/faq-erros-comuns.txt" | jq

# 3. Listar documentos indexados
curl http://localhost:8080/api/v1/documents | jq

# 4. Ver detalhes de um documento
curl http://localhost:8080/api/v1/documents/a1b2c3d4-e5f6-7890-abcd-ef1234567890 | jq

# 5. Remover um documento
curl -X DELETE http://localhost:8080/api/v1/documents/a1b2c3d4-e5f6-7890-abcd-ef1234567890 | jq
```
