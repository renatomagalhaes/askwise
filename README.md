# AskWise — Chatbot RAG para Base de Conhecimento

> "Pergunte com sabedoria, receba respostas inteligentes."

## O que é o AskWise?

AskWise é uma **Proof of Concept (PoC)** de um chatbot inteligente que responde perguntas
com base em documentos enviados pelo usuário. Ele usa a técnica de **RAG (Retrieval-Augmented
Generation)** para buscar informações relevantes nos documentos e gerar respostas precisas,
como se fosse um atendente sênior com anos de experiência.

## Cenário de Uso

Uma empresa de suporte técnico possui conhecimento espalhado em centenas de arquivos (PDFs,
planilhas, documentações, runbooks) e na cabeça dos atendentes mais experientes. Novos
funcionários levam meses para atingir o nível de um atendente sênior.

O AskWise resolve isso: basta fazer upload dos documentos da empresa e qualquer pessoa pode
fazer perguntas no terminal, recebendo respostas instantâneas equivalentes às de um
profissional com anos de experiência.

## Tech Stack

| Componente       | Tecnologia                    | Motivo                                      |
|------------------|-------------------------------|---------------------------------------------|
| Linguagem        | Go (Golang) 1.26              | Performance, simplicidade, ótimo para CLI    |
| API HTTP         | `net/http` (stdlib)           | Sem dependências externas desnecessárias      |
| Vector Store     | Qdrant                        | DB vetorial robusto, API REST, client Go     |
| Metadata Store   | SQLite                        | Zero config, perfeito para PoC               |
| Embeddings       | OpenAI `text-embedding-3-small` | Qualidade alta, custo baixo                |
| LLM              | OpenAI `gpt-4o-mini`          | Rápido, barato, ótimo para PoC              |
| Logs             | `log/slog` JSON               | Estruturado, stdlib, STDOUT/STDERR          |
| Infra            | Docker Compose                | Tudo roda em containers, zero Go local       |

## Estrutura do Projeto

```
askwise/
├── README.md                    # Este arquivo
├── AGENTS.md                    # Instruções para agentes de IA
├── PLAN.md                      # Plano de implementação com etapas
├── Dockerfile                   # Multi-stage: builder, runtime, dev
├── docker-compose.yml           # app + chat + qdrant
├── Makefile                     # Todos os comandos (make help)
├── api/
│   └── openapi.yaml             # Especificação OpenAPI/Swagger
├── docs/
│   ├── spec/                    # SPEC: Especificações do projeto
│   │   ├── 01-VISAO-GERAL.md
│   │   ├── 02-CENARIO-NEGOCIO.md
│   │   ├── 03-REQUISITOS.md
│   │   └── 04-REGRAS-NEGOCIO.md
│   ├── design/                  # Decisões de arquitetura
│   │   ├── 01-ARQUITETURA.md
│   │   ├── 02-MODELO-DADOS.md
│   │   └── 03-API-DESIGN.md
│   ├── adr/                     # Architecture Decision Records
│   │   ├── 001-docker-first.md
│   │   ├── 002-structured-json-logs.md
│   │   ├── 003-sqlite-metadata.md
│   │   ├── 004-qdrant-vectorstore.md
│   │   └── 005-openai-provider.md
│   └── learn/                   # Material educativo sobre RAG e SDD
│       ├── 01-RAG-EXPLAINED.md
│       ├── 02-SDD-METHODOLOGY.md
│       └── 03-EMBEDDINGS.md
├── cmd/
│   ├── server/                  # API HTTP para upload de documentos
│   │   └── main.go
│   └── chat/                    # CLI interativo para perguntas
│       └── main.go
├── internal/
│   ├── config/                  # Carregamento de configuração (.env)
│   ├── logger/                  # Logger JSON estruturado (slog)
│   ├── document/                # Parsing e processamento de documentos
│   ├── chunker/                 # Divisão de texto em chunks
│   ├── embedding/               # Geração de embeddings via OpenAI
│   ├── vectorstore/             # Integração com Qdrant
│   ├── retriever/               # Busca de contexto relevante
│   ├── llm/                     # Integração com LLM (chat completion)
│   ├── rag/                     # Orquestração do pipeline RAG
│   └── storage/                 # SQLite para metadados
├── go.mod
├── go.sum
└── .env.example                 # Variáveis de ambiente
```

## Metodologia: SDD (Spec-Driven Development)

Este projeto segue a metodologia **SDD** — onde a **especificação dirige todo o
desenvolvimento**. Em vez de codificar direto, primeiro criamos documentos de spec
detalhados que servem como fonte de verdade para humanos e agentes de IA:

1. **Spec** — Documentos detalhados definem requisitos, regras, arquitetura, APIs e modelos
2. **Development** — O código é implementado **dirigido pela spec**, usando-a como guia único

A spec não é apenas documentação — ela é o **motor do desenvolvimento**, especialmente
quando combinada com agentes de IA que leem a spec e geram código alinhado automaticamente.

Leia mais em [`docs/learn/02-SDD-METHODOLOGY.md`](docs/learn/02-SDD-METHODOLOGY.md).

## Pré-requisitos

Apenas dois programas são necessários. **Não é preciso instalar Go localmente** — tudo
roda dentro de containers Docker ([ADR-001](docs/adr/001-docker-first.md)).

- [Docker](https://docs.docker.com/get-docker/) >= 24.0
- [Make](https://www.gnu.org/software/make/) (já incluso no macOS e na maioria dos Linux)

## Quick Start

```bash
# 1. Clone o repositório
git clone https://github.com/renatomagalhaes/askwise.git
cd askwise

# 2. Configure as variáveis de ambiente
cp .env.example .env
# Edite .env com sua OPENAI_API_KEY

# 3. Suba toda a infra (compila + qdrant + app)
make up

# 4. Verifique se está tudo saudável
make health

# 5. Faça upload de um documento
make upload FILE=meu-documento.pdf

# 6. Abra o chat interativo
make chat
```

### Comandos Disponíveis

```bash
make help              # Lista todos os comandos
make up                # Sobe tudo (qdrant + app)
make down              # Derruba tudo
make chat              # Abre o chat interativo
make test              # Roda testes unitários
make test-integration  # Roda testes de integração
make logs              # Mostra logs JSON (follow)
make upload FILE=x.pdf # Upload de documento
make health            # Health check da API
make dev-shell         # Shell dentro do container
make clean             # Remove tudo (containers + volumes + dados)
```

## Formatos Suportados

| Formato | Extensão | Descrição                          |
|---------|----------|------------------------------------|
| PDF     | `.pdf`   | Documentos, manuais, runbooks      |
| CSV     | `.csv`   | Planilhas, dados tabulares          |
| TXT     | `.txt`   | Texto puro, notas, logs            |
| YAML    | `.yaml`  | Configurações, playbooks            |
| JSON    | `.json`  | Dados estruturados, configs         |
| Markdown| `.md`    | Documentação, wikis, READMEs        |

## Licença

MIT License — uso livre para aprendizado e experimentação.
