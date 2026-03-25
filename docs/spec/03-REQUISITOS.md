# Requisitos — AskWise

## 1. Requisitos Funcionais (RF)

### RF-01: Upload de Documentos
- **Descrição**: O sistema deve aceitar upload de arquivos via API REST (POST)
- **Formatos**: PDF, CSV, TXT, YAML, JSON, MD
- **Limite**: Arquivos de até 10MB
- **Resposta**: Retornar ID do documento, número de chunks gerados e status
- **Validação**: Rejeitar formatos não suportados com mensagem clara de erro

### RF-02: Extração de Texto
- **Descrição**: O sistema deve extrair texto legível de cada formato suportado
- **PDF**: Extrair texto de todas as páginas (sem OCR nesta PoC)
- **CSV**: Converter linhas em texto estruturado (cabeçalho + valores)
- **TXT**: Usar conteúdo diretamente
- **YAML**: Converter estrutura em texto legível
- **JSON**: Converter estrutura em texto legível
- **MD**: Usar conteúdo diretamente (remover sintaxe markdown se necessário)

### RF-03: Chunking de Documentos
- **Descrição**: O sistema deve dividir o texto extraído em pedaços menores (chunks)
- **Tamanho**: Chunks de ~500 tokens com overlap de ~50 tokens
- **Preservação**: Respeitar limites de parágrafos e seções quando possível
- **Metadados**: Cada chunk deve manter referência ao documento original e posição

### RF-04: Geração de Embeddings
- **Descrição**: O sistema deve gerar embeddings vetoriais para cada chunk
- **Modelo**: OpenAI `text-embedding-3-small` (1536 dimensões)
- **Batch**: Processar múltiplos chunks em uma única chamada API quando possível

### RF-05: Armazenamento de Vetores
- **Descrição**: O sistema deve armazenar embeddings em banco de dados vetorial
- **Busca**: Suportar busca por similaridade (cosine similarity)
- **Metadados**: Armazenar junto: doc_id, chunk_index, texto original, nome do arquivo

### RF-06: Chat via Terminal (CLI)
- **Descrição**: O sistema deve oferecer interface de chat interativo no terminal
- **Entrada**: Texto livre digitado pelo usuário
- **Saída**: Resposta gerada pela LLM com base no contexto recuperado
- **Fontes**: Mostrar quais documentos foram usados para gerar a resposta
- **Histórico**: Manter contexto da conversa durante a sessão (últimas N mensagens)
- **Comandos**: Suportar comandos especiais (/help, /clear, /quit, /sources)

### RF-07: Pipeline RAG
- **Descrição**: O sistema deve implementar o pipeline completo de RAG
- **Retrieval**: Buscar os top-K chunks mais relevantes para a pergunta
- **Augmentation**: Montar prompt com contexto recuperado + pergunta do usuário
- **Generation**: Enviar prompt para LLM e retornar resposta

### RF-08: Listagem de Documentos
- **Descrição**: O sistema deve permitir listar documentos indexados via API
- **Informações**: ID, nome, formato, data de upload, número de chunks, tamanho

### RF-09: Remoção de Documentos
- **Descrição**: O sistema deve permitir remover documentos e seus chunks via API
- **Efeito**: Remover do vector store e da base de metadados

## 2. Requisitos Não-Funcionais (RNF)

### RNF-01: Performance
- Tempo de resposta do chat: < 10 segundos (incluindo chamada à LLM)
- Tempo de upload e processamento: < 30 segundos para arquivos de até 5MB
- Busca vetorial: < 500ms para recuperar top-K chunks

### RNF-02: Código Educativo
- Todo o código deve estar comentado explicando o que faz e por quê
- Nomes de variáveis e funções devem ser descritivos
- Cada pacote deve ter um doc.go explicando sua responsabilidade
- Erros devem ter mensagens claras e contextuais

### RNF-03: Simplicidade
- Mínimo de dependências externas
- Usar standard library do Go sempre que possível
- Configuração via variáveis de ambiente (.env)
- Docker Compose para subir dependências (Qdrant)

### RNF-04: Observabilidade
- Logs estruturados indicando cada etapa do pipeline
- Mostrar no terminal quantos chunks foram encontrados e de quais documentos
- Em caso de erro, mensagens devem indicar a causa e possível solução

### RNF-05: Segurança (Básica para PoC)
- Validar tipo e tamanho de arquivo no upload
- Sanitizar nomes de arquivo
- Não expor informações sensíveis nos logs
- API key da OpenAI via variável de ambiente (nunca hardcoded)

### RNF-06: Portabilidade
- Funcionar em macOS, Linux e Windows
- Go 1.22+
- Docker para dependências externas
- Sem dependência de serviços cloud (exceto OpenAI API)

## 3. Requisitos de Interface

### API REST (Upload)

```
POST   /api/v1/documents          — Upload de documento
GET    /api/v1/documents          — Listar documentos
GET    /api/v1/documents/:id      — Detalhes de um documento
DELETE /api/v1/documents/:id      — Remover documento

GET    /api/v1/health             — Health check
```

### CLI (Chat)

```
┌─────────────────────────────────────────────────────────┐
│                    AskWise Chat v1.0                     │
│                                                         │
│  Base de conhecimento: 6 documentos, 342 chunks          │
│  Digite /help para ver comandos disponíveis              │
│                                                         │
│─────────────────────────────────────────────────────────│
│                                                         │
│  Você: Como resolver o erro CONNECTION_TIMEOUT_5032?     │
│                                                         │
│  AskWise: O erro CONNECTION_TIMEOUT_5032 ocorre quando   │
│  a conexão com o servidor excede o timeout configurado.  │
│                                                         │
│  Para resolver:                                          │
│  1. Verifique a conectividade de rede                    │
│  2. Aumente o timeout em config.yaml (padrão: 30s)       │
│  3. Verifique se o serviço de destino está ativo          │
│                                                         │
│  📄 Fontes:                                              │
│  - runbook-cloudapi.md (chunk 12)                        │
│  - faq-erros-comuns.txt (chunk 7)                        │
│                                                         │
│  Você: _                                                 │
└─────────────────────────────────────────────────────────┘
```

## 4. Restrições Técnicas

| Restrição                | Detalhe                                          |
|--------------------------|--------------------------------------------------|
| Linguagem                | Go (Golang) 1.22+                                |
| LLM Provider             | OpenAI API (requer API key)                       |
| Vector Store             | Qdrant (via Docker)                               |
| Formato de chunks        | Texto puro (sem imagens ou tabelas complexas)     |
| Concorrência             | Single-user (PoC, sem controle de concorrência)   |
| Persistência de chat     | Apenas em memória (durante a sessão)              |

## 5. Dependências Externas

| Dependência              | Versão    | Propósito                              |
|--------------------------|-----------|----------------------------------------|
| Go                       | >= 1.22   | Linguagem principal                    |
| Docker                   | >= 24.0   | Executar Qdrant                        |
| Qdrant                   | >= 1.9    | Banco de dados vetorial                |
| OpenAI API               | v1        | Embeddings + Chat Completion           |
