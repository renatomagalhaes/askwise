# Visão Geral do Projeto AskWise

## 1. Propósito

O AskWise é um sistema de chatbot inteligente que utiliza **RAG (Retrieval-Augmented Generation)**
para responder perguntas com base em documentos fornecidos pelo usuário. O objetivo principal
é transformar conhecimento disperso em respostas instantâneas e precisas.

## 2. Problema

Empresas de suporte técnico enfrentam desafios comuns:

- **Conhecimento fragmentado**: informações espalhadas em PDFs, planilhas, wikis e emails
- **Dependência de pessoas-chave**: apenas atendentes seniores conhecem certas soluções
- **Onboarding lento**: novos funcionários levam meses para se tornarem produtivos
- **Respostas inconsistentes**: diferentes atendentes dão respostas diferentes para o mesmo problema
- **Perda de conhecimento**: quando um funcionário sai, o conhecimento vai junto

## 3. Solução

O AskWise resolve esses problemas em 3 passos:

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   1. UPLOAD  │────▶│  2. PROCESSA │────▶│  3. PERGUNTA │
│              │     │              │     │              │
│  Documentos  │     │  Chunking +  │     │  CLI Chat    │
│  via API     │     │  Embeddings  │     │  com RAG     │
└──────────────┘     └──────────────┘     └──────────────┘
```

1. **Upload**: O usuário envia documentos da empresa via API REST
2. **Processamento**: O sistema divide os documentos em pedaços (chunks), gera embeddings
   vetoriais e armazena no banco de dados vetorial
3. **Consulta**: O usuário faz perguntas no terminal CLI, o sistema busca os trechos mais
   relevantes e gera uma resposta contextualizada usando LLM

## 4. Público-Alvo

### Usuários Diretos
- **Atendentes de suporte** (todos os níveis) — fazem perguntas ao chatbot
- **Gestores de conhecimento** — fazem upload de documentos da base

### Beneficiários Indiretos
- **Clientes da empresa** — recebem respostas mais rápidas e consistentes
- **Novos funcionários** — aceleram o onboarding com acesso ao conhecimento consolidado

## 5. Objetivos de Aprendizado

Como este é um projeto PoC educativo, os objetivos técnicos incluem:

| # | Objetivo                                  | Conceito Aprendido                        |
|---|-------------------------------------------|-------------------------------------------|
| 1 | Implementar upload e parsing de arquivos  | I/O, parsing multi-formato em Go          |
| 2 | Dividir documentos em chunks              | Chunking strategies para RAG              |
| 3 | Gerar embeddings vetoriais                | Text embeddings, representação semântica  |
| 4 | Armazenar e buscar vetores                | Vector databases, similarity search       |
| 5 | Construir pipeline RAG                    | Retrieval-Augmented Generation            |
| 6 | Integrar com LLM                          | Prompt engineering, chat completion       |
| 7 | Criar CLI interativo                      | Terminal UI em Go                         |
| 8 | Seguir metodologia SDD                    | Spec-Design-Development                   |

## 6. Escopo

### Incluído (MVP)
- Upload de documentos via API POST (PDF, CSV, TXT, YAML, JSON, MD)
- Processamento e indexação automática
- Chat via terminal CLI
- Respostas baseadas no conteúdo dos documentos
- Citação das fontes utilizadas na resposta

### Fora do Escopo (Futuro)
- Interface web (frontend)
- Autenticação e autorização
- Multi-tenancy (múltiplas empresas)
- Upload em lote (batch)
- Atualização incremental de documentos
- Deploy em produção / cloud

## 7. Métricas de Sucesso

Para uma PoC, consideramos sucesso quando:

- [ ] É possível fazer upload de pelo menos 3 tipos de arquivo diferentes
- [ ] O chatbot responde perguntas sobre o conteúdo dos documentos enviados
- [ ] As respostas incluem referência ao documento fonte
- [ ] O tempo de resposta é inferior a 10 segundos
- [ ] O código está comentado e documentado para fins de aprendizado
