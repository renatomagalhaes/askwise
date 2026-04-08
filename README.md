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

## Como Funciona (TL;DR)

```
 Seus arquivos                 AskWise                        Você pergunta
 ============          =======================              ================

  manual.pdf    ──┐     1. Extrai o texto do arquivo
  erros.txt     ──┼──►  2. Divide em pedaços pequenos
  tickets.csv   ──┤     3. Transforma cada pedaço em vetor numérico (embedding)
  config.yaml   ──┘     4. Salva os vetores no Qdrant (banco vetorial)
                                    │
                                    ▼
                         Documentos indexados ✓
                                    │
  "Como resolver             ┌──────┘
   o erro 5032?"  ──────►    │
                             ├──  5. Transforma a pergunta em vetor
                             ├──  6. Busca os pedaços mais parecidos no Qdrant
                             ├──  7. Monta um prompt com o contexto encontrado
                             └──  8. Envia para a LLM (GPT) gerar a resposta
                                           │
                                           ▼
                              "O erro 5032 ocorre quando..."
                              📎 Fontes: manual.pdf, erros.txt
```

**Resumo em uma frase:** você junta seus arquivos de conhecimento (PDF, TXT, CSV, YAML, JSON, MD),
envia via API REST, e depois faz perguntas no terminal — o sistema busca os trechos mais
relevantes e usa uma IA para gerar uma resposta precisa, sempre citando as fontes.

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
├── Dockerfile                   # Multi-stage: builder, runtime, dev
├── docker-compose.yml           # app + chat + qdrant
├── Makefile                     # Todos os comandos (make help)
├── api/
│   └── openapi.yaml             # Especificação OpenAPI/Swagger
├── docs/
│   ├── project/                 # Gestão do projeto e agentes
│   │   ├── AGENTS.md            # Instruções para agentes de IA
│   │   └── PLAN.md              # Plano de implementação com etapas
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

## Testando com Documentos de Exemplo

O projeto inclui documentos de teste do cenário fictício **TechSupport Ltda.** — uma
empresa de suporte técnico com runbooks, tickets, políticas e configurações.

```bash
# 1. Suba a infra
make up

# 2. Envie todos os documentos de exemplo
make upload FILE=testdata/techsupport-runbook.md
make upload FILE=testdata/erros-conhecidos.txt
make upload FILE=testdata/politicas-suporte.txt
make upload FILE=testdata/tickets-suporte.csv
make upload FILE=testdata/config-servicos.yaml
make upload FILE=testdata/contatos-equipe.json

# 3. Verifique que foram indexados
curl http://localhost:8484/api/v1/documents | jq

# 4. Abra o chat e faça perguntas
make chat
```

### Exemplos de perguntas para testar

Cada pergunta abaixo pode ser respondida com base nos documentos enviados.
Experimente no `make chat`:

**Troubleshooting (runbook + erros conhecidos):**
```
Como resolver o erro 5032 de timeout?
Quais são as causas do erro 4010 de autenticação OAuth?
O que fazer quando o disco fica cheio no servidor de logs?
Como resolver o erro de rate limit na API de pagamentos?
O certificado SSL expirou, qual o procedimento?
O container está sendo encerrado com OOMKilled, o que fazer?
Como resolver problemas com a fila de mensagens do RabbitMQ?
```

**Políticas e processos:**
```
Qual o SLA de resposta para incidentes P1?
Como funciona o processo de escalação de suporte?
Quais são os canais de atendimento disponíveis?
Qual o horário de atendimento para incidentes P2?
Como funciona a comunicação com o cliente durante incidentes?
Quais métricas são usadas para avaliar o time de suporte?
Qual a política de backup e recuperação do banco de dados?
```

**Deploy e operações:**
```
Me explique o processo de deploy em produção passo a passo.
O que fazer se precisar de rollback em produção?
Qual a configuração de memória do backend-api em produção?
Quais jobs rodam no scheduler? Em que horários?
Quais dashboards existem no Grafana?
```

**Tickets e histórico:**
```
O que aconteceu no ticket TK-006 sobre banco corrompido?
Como foi resolvido o problema de notificações duplicadas?
Qual foi a causa do problema de relatórios lentos no dashboard?
Como resolveram as tentativas de login suspeitas da Beta Systems?
Quais tickets foram classificados como P1?
```

**Equipe e contatos:**
```
Quem é responsável pelo suporte L2?
Qual o canal do Slack para emergências?
Quem contatar para problemas com o gateway de pagamento?
Como funciona a rotação de plantão?
Quem é a especialista em banco de dados e performance?
```

**Follow-up (usa histórico da conversa):**
```
você> Como resolver o erro 5032?
       (aguarde a resposta)
você> E se o problema persistir depois disso?
você> Quem devo acionar nesse caso?
```

### Documentos de teste disponíveis

| Arquivo | Formato | Conteúdo |
|---------|---------|----------|
| `techsupport-runbook.md` | Markdown | Runbook com 5 procedimentos de suporte (erros, deploy, FAQ) |
| `erros-conhecidos.txt` | Texto | 6 erros conhecidos com causa raiz e solução passo a passo |
| `politicas-suporte.txt` | Texto | Políticas de atendimento, SLAs, escalação e métricas |
| `tickets-suporte.csv` | CSV | 12 tickets reais resolvidos com descrição e resolução |
| `config-servicos.yaml` | YAML | Configuração de todos os serviços de produção |
| `contatos-equipe.json` | JSON | Equipes, membros, canais de comunicação e contatos |

## Diagramas de Arquitetura

O documento [`docs/DIAGRAMAS.md`](docs/DIAGRAMAS.md) contém diagramas Mermaid renderizáveis no GitHub:

| Diagrama | Tipo | O que mostra |
|----------|------|-------------|
| C4 Contexto | C4 Context | AskWise vs sistemas externos (OpenAI, Qdrant) |
| C4 Containers | C4 Container | API Server, CLI, SQLite, Qdrant |
| Componentes | Graph | Pacotes Go e suas dependências |
| Ingestão | Sequence | Fluxo completo de upload de documento |
| Consulta | Sequence | Fluxo completo de pergunta no chat |
| Validação Upload | Flowchart | Decisões e validações (RN-01 a RN-05) |
| Loop CLI | Flowchart | Interação do chat com comandos |
| Docker Compose | Graph | Containers e volumes da stack |
| Modelo de Dados | ER Diagram | SQLite (documents) + Qdrant (chunks/vetores) |

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
