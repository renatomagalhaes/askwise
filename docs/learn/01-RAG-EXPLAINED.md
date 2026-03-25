# O que é RAG? — Guia Completo para Iniciantes

## 1. Introdução

**RAG** significa **Retrieval-Augmented Generation** (Geração Aumentada por Recuperação).
É uma técnica que combina busca de informações com geração de texto por IA para produzir
respostas mais precisas e fundamentadas.

### O Problema que o RAG Resolve

Modelos de linguagem (LLMs) como GPT-4, Claude e Gemini são treinados com dados até uma
certa data e não conhecem informações específicas da sua empresa. Quando você pergunta
algo que não está no treinamento deles, eles podem:

1. **Alucinar** — inventar uma resposta convincente mas falsa
2. **Ser genéricos** — dar uma resposta vaga que não ajuda
3. **Recusar** — dizer que não sabem (o melhor dos cenários ruins)

### A Solução: RAG

RAG resolve isso em 3 passos:

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  RETRIEVAL  │───▶│ AUGMENTATION│───▶│ GENERATION  │
│             │    │             │    │             │
│  Buscar     │    │  Montar     │    │  LLM gera   │
│  informação │    │  prompt com │    │  resposta    │
│  relevante  │    │  contexto   │    │  baseada no  │
│  na base    │    │  encontrado │    │  contexto    │
└─────────────┘    └─────────────┘    └─────────────┘
```

## 2. Analogia: O Bibliotecário Inteligente

Imagine que a LLM é um escritor muito talentoso, mas que não leu todos os livros do mundo.

**Sem RAG:**
> Você: "Qual é o procedimento para resetar o cache do CloudAPI?"
> LLM (escritor): "Hmm, não sei especificamente, mas geralmente caches são resetados
> reiniciando o serviço..." (resposta genérica, possivelmente errada)

**Com RAG:**
> Você: "Qual é o procedimento para resetar o cache do CloudAPI?"
> RAG (bibliotecário): "Deixa eu buscar nos manuais..."
> *encontra o runbook-cloudapi.md, seção 4.2*
> RAG: "Encontrei! Aqui está o procedimento relevante."
> LLM (escritor): "Segundo o runbook do CloudAPI, seção 4.2, siga estes passos:
> 1. Acesse o painel admin...
> 2. Navegue para Cache Management...
> 3. Clique em Purge All..."

O RAG funciona como um **bibliotecário** que encontra os livros certos antes de pedir
ao escritor que formule a resposta.

## 3. Os Três Pilares do RAG

### 3.1 Retrieval (Recuperação)

É o processo de buscar informações relevantes em uma base de conhecimento.

**Como funciona:**
1. O texto dos documentos é convertido em vetores numéricos (embeddings)
2. A pergunta do usuário também é convertida em vetor
3. Busca-se os vetores mais similares (cosine similarity)
4. Os textos correspondentes aos vetores encontrados são recuperados

```
Pergunta: "Como resolver erro 5032?"
                │
                ▼
    Gera embedding da pergunta
    [0.012, -0.034, 0.056, ...]
                │
                ▼
    Busca vetores similares no banco
                │
                ▼
    Encontra 5 chunks relevantes:
    1. "Erro 5032: Connection timeout..." (score: 0.92)
    2. "Troubleshooting de timeouts..." (score: 0.87)
    3. "Configuração de timeout..." (score: 0.81)
    4. "Logs de erro comuns..." (score: 0.74)
    5. "Rede e conectividade..." (score: 0.68)
```

### 3.2 Augmentation (Aumento/Enriquecimento)

É a construção do prompt que será enviado à LLM, combinando:
- Instrução do sistema (system prompt)
- Contexto recuperado (chunks relevantes)
- Histórico da conversa
- Pergunta do usuário

**Exemplo de prompt montado:**

```
[System]
Você é um assistente de suporte técnico da TechSupport Ltda.
Responda APENAS com base no contexto fornecido abaixo.
Se a informação não estiver no contexto, diga que não encontrou.

[Contexto Recuperado]
--- Fonte: runbook-cloudapi.md (chunk 8) ---
O erro CONNECTION_TIMEOUT_5032 ocorre quando a conexão com o servidor
excede o timeout configurado. Causas comuns:
1. Firewall bloqueando a porta 443
2. Servidor de destino indisponível
3. Timeout configurado muito baixo (padrão: 30s)
Solução: verificar conectividade, aumentar timeout em config.yaml...

--- Fonte: faq-erros-comuns.txt (chunk 3) ---
Erro 5032 - Timeout de Conexão
Frequência: Alta (top 5 erros mais reportados)
Resolução média: 5 minutos
Passos: ver runbook-cloudapi.md seção 4.2

[Pergunta do Usuário]
Como resolver o erro 5032?
```

### 3.3 Generation (Geração)

A LLM recebe o prompt enriquecido e gera uma resposta que é:
- **Baseada em fatos** — usa o contexto fornecido, não inventa
- **Bem formatada** — organiza a informação de forma clara
- **Com fontes** — cita de onde veio a informação

## 4. Conceitos-Chave

### 4.1 Embeddings (Representações Vetoriais)

Embeddings são representações numéricas de texto em um espaço multidimensional.
Textos com significado semelhante ficam próximos nesse espaço.

```
"gato"     → [0.2, 0.8, 0.1, ...]  ─┐
"felino"   → [0.3, 0.7, 0.1, ...]  ─┤ Próximos (similares)
"gatinho"  → [0.2, 0.9, 0.2, ...]  ─┘

"carro"    → [0.9, 0.1, 0.7, ...]  ── Distante (diferente)
```

Veja mais detalhes em [03-EMBEDDINGS.md](03-EMBEDDINGS.md).

### 4.2 Chunks (Pedaços de Texto)

Documentos inteiros são muito grandes para embeddings e busca eficiente.
Por isso, dividimos em pedaços menores chamados **chunks**.

```
Documento (10.000 palavras)
│
├── Chunk 1 (500 palavras): "Introdução ao CloudAPI..."
├── Chunk 2 (500 palavras): "Configuração inicial..."
├── Chunk 3 (500 palavras): "Erros comuns e soluções..."
├── ...
└── Chunk 20 (500 palavras): "Referências e contatos..."
```

**Por que chunking é importante:**
- Embeddings capturam melhor o significado de textos menores
- Busca retorna trechos específicos, não documentos inteiros
- O contexto no prompt fica mais focado e relevante

### 4.3 Vector Store (Banco de Dados Vetorial)

É um banco de dados otimizado para armazenar e buscar vetores de alta dimensão.

**Analogia**: Se um banco SQL é como um catálogo organizado por categorias,
um vector store é como uma biblioteca onde os livros estão organizados por
**similaridade de conteúdo** — livros sobre temas parecidos ficam na mesma estante.

**Operações principais:**
- **Upsert**: Inserir/atualizar vetores
- **Search**: Buscar os K vetores mais similares a um vetor de consulta
- **Delete**: Remover vetores
- **Filter**: Filtrar por metadados (ex: só buscar em PDFs)

### 4.4 Cosine Similarity (Similaridade de Cosseno)

Métrica que mede o ângulo entre dois vetores. Quanto menor o ângulo, mais similares.

```
Score 1.0  → Idênticos (mesmo significado)
Score 0.8+ → Muito similares (mesmo tema)
Score 0.5  → Alguma relação
Score 0.0  → Sem relação
Score -1.0 → Opostos
```

No AskWise, usamos threshold de 0.5 — só chunks com score >= 0.5 são considerados
relevantes.

## 5. Pipeline RAG Completo

```
┌────────────────── FASE DE INGESTÃO (Upload) ──────────────────┐
│                                                                │
│  Documento ──▶ Parser ──▶ Texto ──▶ Chunker ──▶ []Chunks     │
│                                                    │           │
│                                          Embedder  │           │
│                                            │       │           │
│                                            ▼       ▼           │
│                                     []Vectors + []Chunks      │
│                                            │                   │
│                                            ▼                   │
│                                     Vector Store (Qdrant)      │
│                                                                │
└────────────────────────────────────────────────────────────────┘

┌────────────────── FASE DE CONSULTA (Chat) ────────────────────┐
│                                                                │
│  Pergunta ──▶ Embedder ──▶ Query Vector                       │
│                                │                               │
│                                ▼                               │
│                     Vector Store.Search(query, top_k=5)        │
│                                │                               │
│                                ▼                               │
│                     Top 5 chunks relevantes                    │
│                                │                               │
│                                ▼                               │
│         ┌──────────────────────────────────────────┐          │
│         │  Prompt = System + Contexto + Pergunta    │          │
│         └──────────────────────┬───────────────────┘          │
│                                │                               │
│                                ▼                               │
│                         LLM (GPT-4o-mini)                      │
│                                │                               │
│                                ▼                               │
│                     Resposta + Fontes citadas                  │
│                                                                │
└────────────────────────────────────────────────────────────────┘
```

## 6. Vantagens e Limitações do RAG

### Vantagens

| Vantagem                     | Explicação                                          |
|------------------------------|-----------------------------------------------------|
| Respostas fundamentadas      | Baseadas em documentos reais, não imaginação da LLM |
| Atualização fácil            | Basta adicionar/remover documentos                  |
| Sem re-treinamento           | Não precisa treinar a LLM (caro e complexo)         |
| Citação de fontes            | O usuário pode verificar a resposta                 |
| Custo baixo                  | Embedding é barato, LLM pequena é suficiente        |
| Privacidade                  | Dados ficam no seu banco, não no treinamento da LLM |

### Limitações

| Limitação                    | Explicação                                          |
|------------------------------|-----------------------------------------------------|
| Qualidade do chunking        | Chunks mal divididos geram respostas ruins          |
| Limite de contexto           | LLMs têm limite de tokens no prompt                 |
| Latência                     | Busca + LLM adiciona alguns segundos                |
| Dependência da base          | Se o doc não foi enviado, a resposta não existe     |
| Documentos complexos         | Tabelas, imagens e diagramas são difíceis de parsear|

## 7. RAG vs Alternativas

| Abordagem         | Quando usar                                | Custo    | Complexidade |
|--------------------|--------------------------------------------|----------|--------------|
| **RAG**            | Base de conhecimento dinâmica              | Baixo    | Média        |
| Fine-tuning        | Estilo/tom específico, domínio técnico     | Alto     | Alta         |
| Prompt engineering | Poucos dados, informação estática          | Muito baixo | Baixa     |
| Knowledge graph    | Relações complexas entre entidades         | Alto     | Muito alta   |

Para o cenário do AskWise (base de suporte com documentos), **RAG é a abordagem ideal**.

## 8. Leitura Adicional

- [Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks](https://arxiv.org/abs/2005.11401) — Paper original do RAG (2020)
- [Building RAG Applications](https://www.pinecone.io/learn/retrieval-augmented-generation/) — Guia prático da Pinecone
- [OpenAI Embeddings Guide](https://platform.openai.com/docs/guides/embeddings) — Documentação oficial
- [Qdrant Documentation](https://qdrant.tech/documentation/) — Documentação do vector store
