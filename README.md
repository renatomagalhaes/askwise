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
| Linguagem        | Go (Golang)                   | Performance, simplicidade, ótimo para CLI    |
| API HTTP         | `net/http` (stdlib)           | Sem dependências externas desnecessárias      |
| Vector Store     | Qdrant                        | DB vetorial robusto, API REST, client Go     |
| Metadata Store   | SQLite                        | Zero config, perfeito para PoC               |
| Embeddings       | OpenAI `text-embedding-3-small` | Qualidade alta, custo baixo                |
| LLM              | OpenAI `gpt-4o-mini`          | Rápido, barato, ótimo para PoC              |
| Infra            | Docker Compose                | Sobe Qdrant com um comando                  |

## Estrutura do Projeto

```
askwise/
├── README.md                    # Este arquivo
├── AGENTS.md                    # Instruções para agentes de IA
├── docs/
│   ├── spec/                    # SPEC: Especificações do projeto
│   │   ├── 01-VISAO-GERAL.md
│   │   ├── 02-CENARIO-NEGOCIO.md
│   │   ├── 03-REQUISITOS.md
│   │   └── 04-REGRAS-NEGOCIO.md
│   ├── design/                  # DESIGN: Decisões de arquitetura
│   │   ├── 01-ARQUITETURA.md
│   │   ├── 02-MODELO-DADOS.md
│   │   └── 03-API-DESIGN.md
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
│   ├── document/                # Parsing e processamento de documentos
│   ├── chunker/                 # Divisão de texto em chunks
│   ├── embedding/               # Geração de embeddings via OpenAI
│   ├── vectorstore/             # Integração com Qdrant
│   ├── retriever/               # Busca de contexto relevante
│   ├── llm/                     # Integração com LLM (chat completion)
│   ├── rag/                     # Orquestração do pipeline RAG
│   └── storage/                 # SQLite para metadados
├── docker-compose.yml           # Qdrant + dependências
├── go.mod
├── go.sum
├── Makefile                     # Comandos úteis
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

## Quick Start

```bash
# 1. Clone o repositório
git clone https://github.com/seu-usuario/askwise.git
cd askwise

# 2. Configure as variáveis de ambiente
cp .env.example .env
# Edite .env com sua OPENAI_API_KEY

# 3. Suba o Qdrant
docker compose up -d

# 4. Inicie o servidor de upload
go run cmd/server/main.go

# 5. Faça upload de um documento (em outro terminal)
curl -X POST http://localhost:8080/api/v1/documents \
  -F "file=@meu-documento.pdf"

# 6. Inicie o chat
go run cmd/chat/main.go
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
