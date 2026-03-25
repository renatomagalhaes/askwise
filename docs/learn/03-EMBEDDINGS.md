# Embeddings e Vetores — Guia para Iniciantes

## 1. O que são Embeddings?

**Embeddings** são representações numéricas (vetores) de texto que capturam seu
**significado semântico**. Textos com significados similares produzem vetores
próximos no espaço vetorial.

### Analogia: Coordenadas GPS

Pense em cidades em um mapa:
- São Paulo e Campinas estão **perto** (significados similares)
- São Paulo e Tóquio estão **longe** (significados diferentes)

Embeddings fazem a mesma coisa, mas com **texto**:
- "O servidor está fora do ar" e "O sistema caiu" → vetores **próximos**
- "O servidor está fora do ar" e "Receita de bolo" → vetores **distantes**

## 2. Como Embeddings Funcionam

### De Texto para Números

```
Texto: "resetar cache do CloudAPI"
                │
                ▼
     Modelo de Embedding
     (text-embedding-3-small)
                │
                ▼
Vetor: [0.0123, -0.0456, 0.0789, 0.0234, -0.0567, ...]
        \_________________________________________________/
                    1536 números (dimensões)
```

Cada dimensão captura um aspecto do significado:
- Algumas dimensões capturam o **tema** (tecnologia vs. culinária)
- Outras capturam o **sentimento** (problema vs. solução)
- Outras capturam a **especificidade** (genérico vs. técnico)

Na prática, as dimensões não têm significados interpretáveis por humanos —
são aprendidas automaticamente pelo modelo de IA.

### Exemplo Visual (Simplificado)

Imagine embeddings de apenas 2 dimensões (na realidade são 1536):

```
         Tema Técnico ↑
              1.0 │
                  │    ● "erro de timeout"
                  │  ● "servidor caiu"
              0.5 │       ● "resetar cache"
                  │
                  │
              0.0 │─────────────────────────▶ Tema Culinária
                  │          ● "bolo de chocolate"
                  │              ● "receita de pão"
             -0.5 │
                  │
             -1.0 │
```

Textos técnicos ficam agrupados no canto superior esquerdo.
Textos culinários ficam em outra região.

## 3. Cosine Similarity (Similaridade de Cosseno)

### O que é?

É a métrica que usamos para medir **quão similares** dois vetores são.
Ela mede o **ângulo** entre os vetores, não a distância.

```
Score:  1.0 = Mesma direção (muito similares)
        0.0 = Perpendiculares (sem relação)
       -1.0 = Direções opostas (opostos)
```

### Fórmula

```
                    A · B           Σ(Ai × Bi)
cos(θ) = ─────────────────── = ───────────────────
              ||A|| × ||B||     √Σ(Ai²) × √Σ(Bi²)
```

### Exemplo Prático

```
Vetor A: "erro de conexão"  = [0.8, 0.6]
Vetor B: "timeout de rede"  = [0.7, 0.7]
Vetor C: "receita de bolo"  = [-0.5, 0.3]

Similaridade A↔B = 0.98  (muito similares — mesmo tema!)
Similaridade A↔C = 0.12  (pouca relação — temas diferentes)
```

### No AskWise

Quando o usuário pergunta "Como resolver timeout?":

1. Geramos embedding da pergunta: `[0.75, 0.65, ...]`
2. Comparamos com TODOS os chunks armazenados
3. Retornamos os top 5 com maior similaridade
4. Filtramos os com score >= 0.5

## 4. Modelos de Embedding

### Comparação de Modelos (OpenAI)

| Modelo                    | Dimensões | Custo/1M tokens | Qualidade |
|---------------------------|-----------|-----------------|-----------|
| `text-embedding-3-small`  | 1536      | $0.02           | Boa       |
| `text-embedding-3-large`  | 3072      | $0.13           | Ótima     |
| `text-embedding-ada-002`  | 1536      | $0.10           | Boa       |

**No AskWise usamos `text-embedding-3-small`** porque:
- Custo muito baixo ($0.02 por milhão de tokens)
- Qualidade suficiente para a PoC
- 1536 dimensões é um bom equilíbrio entre precisão e performance

### Alternativas Open Source

| Modelo                | Provider    | Dimensões | Custo    |
|-----------------------|-------------|-----------|----------|
| all-MiniLM-L6-v2      | HuggingFace | 384       | Grátis   |
| nomic-embed-text       | Ollama      | 768       | Grátis   |
| mxbai-embed-large      | Ollama      | 1024      | Grátis   |

Para a PoC usamos OpenAI pela simplicidade, mas no futuro pode-se trocar
por Ollama para rodar localmente sem custos.

## 5. Chunking — Preparando Texto para Embeddings

### Por que dividir em Chunks?

1. **Limite de tokens**: Modelos de embedding têm limite (~8K tokens)
2. **Precisão**: Textos menores geram embeddings mais específicos
3. **Relevância**: Na busca, retornar um trecho relevante é melhor que um documento inteiro

### Estratégias de Chunking

#### Chunking por Tamanho Fixo
```
Texto: "AAAA BBBB CCCC DDDD EEEE FFFF GGGG HHHH"
Chunk size: 4 palavras

Chunk 1: "AAAA BBBB CCCC DDDD"
Chunk 2: "EEEE FFFF GGGG HHHH"

Problema: pode cortar no meio de uma frase!
```

#### Chunking com Overlap (Usado no AskWise)
```
Texto: "AAAA BBBB CCCC DDDD EEEE FFFF GGGG HHHH"
Chunk size: 4 palavras, Overlap: 2 palavras

Chunk 1: "AAAA BBBB CCCC DDDD"
Chunk 2: "CCCC DDDD EEEE FFFF"  ← overlap com chunk 1
Chunk 3: "EEEE FFFF GGGG HHHH"  ← overlap com chunk 2

Vantagem: contexto preservado nas fronteiras!
```

#### Chunking Recursivo por Separadores (Melhor abordagem)
```
Prioridade de separação:
1. "\n\n" (parágrafos)    ← tenta primeiro
2. "\n"   (linhas)        ← se chunk ainda é grande
3. ". "   (frases)        ← se ainda é grande
4. " "    (palavras)      ← último recurso

Resultado: chunks que respeitam a estrutura do texto
```

### Parâmetros no AskWise

```
chunk_size    = 500 tokens  (~375 palavras)
chunk_overlap = 50 tokens   (~38 palavras)
separators    = ["\n\n", "\n", ". ", " "]
```

## 6. Vector Store — Onde Vetores Moram

### O que é?

Um **vector store** (banco de dados vetorial) é otimizado para:
- Armazenar vetores de alta dimensão (ex: 1536 floats)
- Buscar os K vetores mais similares a um vetor de consulta (KNN)
- Filtrar por metadados associados aos vetores

### Como a Busca Funciona (Simplificado)

```
Sua pergunta: "Como resolver timeout?"
Embedding:    [0.75, 0.65, 0.12, ...]

Vector Store faz (internamente):
1. Compara com TODOS os vetores armazenados
2. Calcula cosine similarity com cada um
3. Ordena por score (maior = mais similar)
4. Retorna top K resultados

Resultado:
┌──────────────────────────────────────────┬───────┐
│ Chunk                                     │ Score │
├──────────────────────────────────────────┼───────┤
│ "Erro timeout 5032: verificar rede..."    │ 0.92  │
│ "Timeout configs em config.yaml..."       │ 0.87  │
│ "Problemas de conexão: timeout..."        │ 0.81  │
│ "Logs de erro: TimeoutException..."       │ 0.74  │
│ "Monitoramento de latência..."            │ 0.68  │
└──────────────────────────────────────────┴───────┘
```

### Qdrant no AskWise

Usamos **Qdrant** como vector store. Ele roda em Docker e expõe uma API REST:

```bash
# Subir Qdrant
docker compose up -d

# Verificar se está rodando
curl http://localhost:6333/healthz
```

**Por que Qdrant:**
- API REST simples
- Client Go disponível
- Roda local via Docker (sem custos)
- Suporta filtros por metadados
- Boa documentação

## 7. Fluxo Completo: Da Pergunta à Resposta

```
Pergunta: "Como resetar o cache do CloudAPI?"
│
├─ 1. Embedding da pergunta
│     [0.71, -0.23, 0.45, ...] (1536 dims)
│
├─ 2. Busca no Qdrant (top 5)
│     ┌─────────────────────────────────────────────────┐
│     │ Score 0.94: "Para resetar cache: 1) Acesse..."  │ ← runbook-cloudapi.md
│     │ Score 0.88: "Cache Management: O CloudAPI..."    │ ← manual-cloudapi.pdf
│     │ Score 0.79: "Erro de cache: quando o cache..."   │ ← faq-erros.txt
│     │ Score 0.65: "Configuração: cache_ttl=3600..."    │ ← config-cloudapi.yaml
│     │ Score 0.52: "Manutenção preventiva: limpar..."   │ ← runbook-datasync.md
│     └─────────────────────────────────────────────────┘
│
├─ 3. Monta prompt com contexto
│     System: "Você é um assistente de suporte..."
│     Contexto: [5 chunks acima]
│     Pergunta: "Como resetar o cache do CloudAPI?"
│
├─ 4. LLM gera resposta
│     "Para resetar o cache do CloudAPI, siga estes passos:
│      1. Acesse o painel de administração
│      2. Navegue para Settings > Cache Management
│      3. Clique em 'Purge All Cache'
│      4. Aguarde a confirmação..."
│
└─ 5. Exibe resposta + fontes
      Fontes: runbook-cloudapi.md, manual-cloudapi.pdf
```

## 8. Custos Estimados

Para a PoC do AskWise com ~50 documentos:

| Operação            | Volume            | Custo Estimado         |
|---------------------|-------------------|------------------------|
| Embedding upload    | ~100K tokens      | ~$0.002 (0.2 centavos) |
| Embedding queries   | ~1K queries/mês   | ~$0.001                |
| Chat completion     | ~1K queries/mês   | ~$0.50                 |
| Qdrant              | Local (Docker)    | Grátis                 |
| **Total/mês**       |                   | **~$0.50**             |

RAG é uma das abordagens mais custo-eficientes para dar conhecimento a uma LLM.

## 9. Termos-Chave (Glossário)

| Termo              | Definição                                                       |
|--------------------|-----------------------------------------------------------------|
| Embedding          | Representação vetorial numérica de texto                        |
| Vetor              | Lista ordenada de números (ex: [0.1, 0.2, 0.3])                |
| Dimensão           | Quantidade de números no vetor (1536 no nosso caso)             |
| Cosine Similarity  | Métrica de similaridade entre vetores (0 a 1)                   |
| Chunk              | Pedaço de texto de um documento                                 |
| Token              | Unidade de texto (~0.75 palavras em português)                  |
| Vector Store       | Banco de dados otimizado para busca de vetores                  |
| KNN                | K-Nearest Neighbors — busca dos K vizinhos mais próximos        |
| Top-K              | Os K resultados mais relevantes de uma busca                    |
| Payload            | Metadados associados a um vetor (texto, nome do arquivo, etc.)  |
