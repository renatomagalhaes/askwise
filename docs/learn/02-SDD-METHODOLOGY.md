# Metodologia SDD — Spec-Driven Development

## 1. O que é SDD?

**SDD (Spec-Driven Development)** é uma metodologia onde a **especificação dirige todo o
desenvolvimento**. A spec não é apenas documentação que ninguém lê — ela é a **fonte única
de verdade** que guia cada decisão de código, arquitetura e implementação.

O nome diz tudo: **Spec-Driven** — a especificação é o motor, o volante, o GPS.
O código é o carro que segue a direção definida pela spec.

```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│              SPEC  (Fonte Única de Verdade)                   │
│                                                              │
│   Requisitos ─── Regras ─── Arquitetura ─── APIs ─── Dados  │
│                                                              │
└──────────┬──────────────────────┬────────────────────────────┘
           │                      │
           │    DIRIGE            │    DIRIGE
           ▼                      ▼
    ┌──────────────┐      ┌──────────────┐
    │  Desenvolvedor│      │  Agente IA   │
    │  (humano)    │      │  (Cursor,    │
    │              │      │   Copilot)   │
    │  Lê a spec,  │      │  Lê a spec,  │
    │  implementa  │      │  gera código │
    │  alinhado    │      │  alinhado    │
    └──────────────┘      └──────────────┘
```

### Analogia: O GPS da Viagem

Imagine uma viagem de carro:

- **Sem SDD**: Você sai dirigindo "mais ou menos pra lá", fazendo curvas erradas,
  voltando, perguntando no caminho. Chega (talvez), mas gasta o triplo do tempo.
- **Com SDD**: Você programa o GPS (spec) com destino exato, paradas planejadas e
  rotas alternativas. O GPS **dirige** suas decisões a cada cruzamento.

A spec é o GPS. Você (ou a IA) é o motorista.

## 2. Por que SDD é Essencial com IA?

SDD ganha um poder multiplicado quando usado com agentes de IA (Cursor, Copilot, ChatGPT).
Esse é o contexto central do nosso projeto AskWise.

### O Problema: IA sem Spec

```
Você: "Cria um sistema de upload de arquivos em Go"

IA: *gera código genérico*
    *inventa nomes de variáveis*
    *escolhe bibliotecas arbitrárias*
    *não segue nenhum padrão*
    *cada pergunta gera código inconsistente com o anterior*
```

Resultado: código fragmentado, sem coesão, que você precisa reescrever.

### A Solução: IA Dirigida por Spec

```
Você: "Implemente o internal/chunker seguindo:
       - Spec: docs/spec/03-REQUISITOS.md (RF-03)
       - Spec: docs/spec/04-REGRAS-NEGOCIO.md (RN-06 a RN-09)
       - Design: docs/design/01-ARQUITETURA.md (seção 2.4)"

IA: *lê os documentos referenciados*
    *entende chunk_size=500, overlap=50*
    *vê a interface Chunker definida*
    *sabe os separadores: \n\n, \n, ". ", " "*
    *implementa EXATAMENTE o que foi especificado*
```

Resultado: código preciso, consistente, rastreável à spec.

### A Diferença em Números

| Métrica                         | Sem SDD | Com SDD  |
|---------------------------------|---------|----------|
| Iterações até código correto    | 5-10    | 1-2      |
| Consistência entre componentes  | Baixa   | Alta     |
| Retrabalho                      | 60%     | 10%      |
| Tempo para onboarding de novo dev/IA | Alto | Baixo |
| Código alinhado com requisitos  | ~40%    | ~95%     |

## 3. Os Pilares do SDD

### Pilar 1: Spec como Fonte Única de Verdade

Tudo começa e termina na spec. Ela contém:

| Documento                | O que Define                              | Quem Consulta       |
|--------------------------|-------------------------------------------|----------------------|
| Visão Geral              | Problema, solução, escopo, métricas       | Todos                |
| Cenário de Negócio       | Contexto real, personas, jornadas         | Todos                |
| Requisitos (RF/RNF)      | O que o sistema faz e como se comporta    | Devs, IA             |
| Regras de Negócio (RN)   | Lógica obrigatória do sistema             | Devs, IA             |
| Arquitetura              | Componentes, interfaces, fluxos           | Devs, IA             |
| Modelo de Dados          | Tabelas, structs, relacionamentos         | Devs, IA             |
| API Design               | Endpoints, payloads, contratos            | Devs, IA, Frontend   |

### Pilar 2: Rastreabilidade (Spec → Código)

Cada linha de código deve ser **rastreável** a um item da spec:

```go
// RF-03 + RN-06: Divide o texto em chunks de ~500 tokens com overlap de ~50 tokens.
// A estratégia de chunking recursivo preserva a estrutura semântica do texto,
// tentando quebrar primeiro por parágrafos, depois por linhas, frases e palavras.
func (c *RecursiveChunker) Chunk(doc *Document) ([]Chunk, error) {
    // RN-07: Overlap entre chunks para preservar contexto nas fronteiras
    // RN-09: Separadores específicos por formato de arquivo
    ...
}
```

Isso permite:
- Saber **por que** cada código existe
- Verificar se a implementação está **completa** (todos os RFs implementados?)
- A IA entender o **propósito** do código ao ler os comentários

### Pilar 3: Spec Dirige a IA

A spec funciona como **prompt engineering estruturado** para agentes de IA:

```
┌─────────────────────────────────────────────────┐
│  Spec Documents (instruções para a IA)           │
│                                                  │
│  "O chunker deve ter chunks de 500 tokens"       │──▶ IA gera chunker com 500 tokens
│  "Overlap de 50 tokens entre chunks"             │──▶ IA implementa overlap de 50
│  "Interface: Chunk(doc) ([]Chunk, error)"        │──▶ IA segue a interface definida
│  "Formatos: PDF, CSV, TXT, YAML, JSON, MD"      │──▶ IA cria parser para cada formato
│                                                  │
└─────────────────────────────────────────────────┘
```

## 4. Fluxo SDD na Prática

### Etapa 1: Escrever a Spec

Antes de qualquer código, documente tudo que o sistema precisa ser:

```
┌─────────────────────────────────────────────────────────────┐
│  SPEC (docs/spec/)                                           │
│                                                              │
│  📄 01-VISAO-GERAL.md      Propósito, problema, escopo      │
│  📄 02-CENARIO-NEGOCIO.md  Personas, jornadas, contexto     │
│  📄 03-REQUISITOS.md       RF-01..RF-09, RNF-01..RNF-06     │
│  📄 04-REGRAS-NEGOCIO.md   RN-01..RN-22                     │
│                                                              │
│  📄 design/01-ARQUITETURA.md   Componentes e interfaces     │
│  📄 design/02-MODELO-DADOS.md  Tabelas, structs, schemas    │
│  📄 design/03-API-DESIGN.md    Endpoints e contratos        │
│                                                              │
└───────────────────────────┬─────────────────────────────────┘
                            │
                     A spec está pronta.
                     Agora ela DIRIGE tudo.
                            │
                            ▼
```

### Etapa 2: Desenvolver Dirigido pela Spec

O desenvolvimento acontece **sempre referenciando a spec**:

```
                            │
           ┌────────────────┼────────────────┐
           │                │                │
           ▼                ▼                ▼
    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
    │  Componente  │ │  Componente  │ │  Componente  │
    │  storage     │ │  chunker     │ │  retriever   │
    │              │ │              │ │              │
    │  Dirigido por│ │  Dirigido por│ │  Dirigido por│
    │  02-MODELO   │ │  RF-03       │ │  RN-10..12   │
    │  RN-21..22   │ │  RN-06..09   │ │  01-ARQUIT.  │
    └──────────────┘ └──────────────┘ └──────────────┘
```

Cada componente sabe **exatamente** quais documentos da spec o dirigem.

### Etapa 3: Validar contra a Spec

Ao finalizar, a spec serve como **checklist de validação**:

- [ ] RF-01 (Upload) → Implementado em `cmd/server`?
- [ ] RF-03 (Chunking) → Implementado em `internal/chunker`?
- [ ] RN-06 (Chunk size 500) → Configurável e testado?
- [ ] RN-11 (Score >= 0.5) → Filtro aplicado no retriever?
- [ ] RN-13 (Resposta baseada em contexto) → System prompt correto?

## 5. SDD vs Outras Metodologias

| Aspecto                  | SDD                     | Agile/Scrum              | Waterfall                |
|--------------------------|-------------------------|--------------------------|--------------------------|
| Filosofia                | Spec dirige tudo        | Iteração contínua        | Fases sequenciais rígidas|
| Documentação             | Essencial e viva        | Mínima e descartável     | Extensiva e pesada       |
| Papel da spec            | Motor do desenvolvimento| Backlog de user stories  | Documento de requisitos  |
| Adequado para IA         | Excelente               | Razoável                 | Razoável                 |
| Flexibilidade            | Spec evolui com o código| Alta                     | Baixa                    |
| Complexidade do processo | Baixa                   | Média                    | Alta                     |
| Ideal para               | PoCs, MVPs, dev com IA  | Produtos em evolução     | Sistemas críticos        |

### Por que SDD supera as outras com IA?

- **Agile**: User stories são vagas demais para a IA ("Como usuário, quero fazer upload").
  A IA precisa de **especificações técnicas precisas**, não narrativas.
- **Waterfall**: Documentação pesada demais, separada do código, desatualizada rapidamente.
  SDD mantém a spec **viva e próxima do código**.
- **SDD**: Specs detalhadas em Markdown, versionadas no Git, legíveis por humanos e IAs.
  A IA lê a spec e gera código preciso. O humano valida contra a mesma spec.

## 6. Dicas para Aplicar SDD

### Para Humanos

1. **Não pule a spec**: A tentação de "ir direto pro código" é grande, mas gera retrabalho
2. **80% é suficiente**: A spec não precisa ser perfeita para começar — ela evolui
3. **Numere tudo**: RF-01, RN-01, RNF-01 — facilita referência cruzada no código
4. **Markdown no Git**: Specs versionadas junto com o código, sempre atualizadas
5. **Referência cruzada**: Comentários no código apontam para itens da spec

### Para Agentes de IA

1. **Sempre referencie a spec**: "Implemente seguindo RF-03 e RN-06 a RN-09"
2. **Inclua os arquivos**: Dê à IA acesso aos documentos da spec como contexto
3. **Valide o output**: Compare o código gerado com o que a spec pede
4. **Itere na spec, não no código**: Se o resultado não ficou bom, melhore a spec primeiro
5. **Use AGENTS.md**: Centralize instruções para a IA sobre como usar a spec

### Regra de Ouro

> **Se a IA gera código que não segue a spec, o problema está na spec, não na IA.**
>
> Melhore a spec → IA gera código melhor → Todos ganham.

## 7. SDD no Contexto de RAG

O AskWise é um projeto que aplica SDD para construir um sistema RAG. Veja como os
conceitos se conectam:

```
┌──────────────────────────────────────────────────────────────────┐
│                        SDD + RAG                                  │
│                                                                   │
│  A SPEC define:                    O RAG implementa:              │
│                                                                   │
│  RF-03: Chunking                   → internal/chunker             │
│  RF-04: Embeddings                 → internal/embedding           │
│  RF-05: Vector store               → internal/vectorstore         │
│  RF-07: Pipeline RAG               → internal/rag                 │
│  RN-10..12: Retrieval              → internal/retriever           │
│  RN-13..17: Generation             → internal/llm                 │
│                                                                   │
│  Cada componente do RAG é DIRIGIDO por um item da spec.           │
│  Nada é inventado. Tudo é rastreável.                             │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

Isso garante que o pipeline RAG não é implementado "no feeling", mas sim seguindo
especificações precisas de tamanho de chunk, score mínimo, número de resultados, tom da
resposta, etc.

## 8. Checklist SDD

### Antes de Codificar
- [ ] A spec está escrita e revisada?
- [ ] Requisitos funcionais cobrem todos os casos?
- [ ] Regras de negócio são claras e sem ambiguidade?
- [ ] Arquitetura e interfaces estão definidas?
- [ ] Modelo de dados e API estão documentados?
- [ ] AGENTS.md está configurado para orientar a IA?

### Durante o Desenvolvimento
- [ ] Cada componente referencia itens da spec nos comentários?
- [ ] A implementação segue as interfaces definidas na spec?
- [ ] Decisões que divergem da spec estão documentadas e justificadas?

### Após o Desenvolvimento
- [ ] Todos os RFs estão implementados?
- [ ] Todas as RNs estão respeitadas?
- [ ] Os RNFs estão sendo atendidos (performance, logs, etc.)?
- [ ] A spec foi atualizada com mudanças feitas durante o desenvolvimento?

## 9. Leitura Adicional

- [Spec-Driven Development: The Future of AI-Assisted Coding](https://www.cursor.com/blog) — Conceito aplicado a dev com IA
- [Documentation-Driven Development](https://gist.github.com/zsup/9434452) — Abordagem similar focada em docs
- [README Driven Development](https://tom.preston-werner.com/2010/08/23/readme-driven-development.html) — Tom Preston-Werner (GitHub)
- [Design Docs at Google](https://www.industrialempathy.com/posts/design-docs-at-google/) — Como Google usa docs para dirigir o dev
