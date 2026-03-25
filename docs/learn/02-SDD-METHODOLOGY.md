# Metodologia SDD — Spec-Design-Development

## 1. O que é SDD?

**SDD (Spec-Design-Development)** é uma metodologia de desenvolvimento que organiza a
construção de software em três fases sequenciais e bem definidas. Cada fase produz
artefatos que alimentam a próxima.

```
┌────────────┐     ┌────────────┐     ┌────────────────┐
│    SPEC    │────▶│   DESIGN   │────▶│  DEVELOPMENT   │
│            │     │            │     │                │
│  O QUE     │     │  COMO      │     │  CONSTRUIR     │
│  construir │     │  construir │     │  de fato       │
└────────────┘     └────────────┘     └────────────────┘
```

### A Analogia da Casa

Pense em construir uma casa:

1. **Spec** = O que o morador quer: "3 quartos, 2 banheiros, garagem, quintal grande"
2. **Design** = A planta do arquiteto: desenhos, materiais, estrutura
3. **Development** = A construção: pedreiro seguindo a planta

Ninguém começa a erguer paredes sem saber quantos quartos terá.
Da mesma forma, não devemos codificar sem saber o que e como construir.

## 2. Por que usar SDD?

### Problemas que SDD Resolve

| Problema Comum                          | Como SDD Resolve                        |
|-----------------------------------------|-----------------------------------------|
| "O que exatamente vamos construir?"     | Spec define claramente escopo e regras  |
| "Cada dev implementa de um jeito"       | Design padroniza arquitetura e APIs     |
| "Descobrimos problemas tarde demais"    | Spec e Design antecipam questões        |
| "O código virou um espaguete"           | Design define estrutura modular         |
| "IA gera código que não faz sentido"    | Spec e Design guiam a geração de código |

### SDD + Desenvolvimento com IA

SDD é especialmente poderoso quando combinado com assistentes de IA (como Cursor, Copilot):

- **Spec** → Dá à IA contexto sobre O QUE construir
- **Design** → Dá à IA contexto sobre COMO construir
- **Development** → IA gera código alinhado com spec e design

Sem SDD, a IA gera código "genérico". Com SDD, a IA gera código **específico e alinhado**.

## 3. Fase 1: SPEC (Especificação)

### O que produzir

A fase de Spec responde: **"O QUE vamos construir e POR QUÊ?"**

| Artefato              | Conteúdo                                          | Exemplo no AskWise             |
|-----------------------|---------------------------------------------------|--------------------------------|
| Visão Geral           | Propósito, problema, solução, escopo              | `01-VISAO-GERAL.md`           |
| Cenário de Negócio    | Contexto real, personas, jornadas                 | `02-CENARIO-NEGOCIO.md`       |
| Requisitos            | Funcionais, não-funcionais, restrições            | `03-REQUISITOS.md`            |
| Regras de Negócio     | Lógica e regras que o sistema deve seguir         | `04-REGRAS-NEGOCIO.md`        |

### Boas Práticas

- **Seja específico**: "O sistema aceita PDF, CSV, TXT" em vez de "O sistema aceita arquivos"
- **Use cenários reais**: Personas e jornadas de uso concretas
- **Defina limites**: O que está no escopo e o que NÃO está
- **Numere tudo**: RF-01, RN-01 facilitam referência cruzada
- **Priorize**: Nem tudo precisa estar no MVP

### Checklist da Spec

- [ ] O problema está claramente descrito?
- [ ] A solução é específica e mensurável?
- [ ] As personas e jornadas estão definidas?
- [ ] Os requisitos funcionais cobrem todos os casos?
- [ ] Os requisitos não-funcionais definem qualidade esperada?
- [ ] As regras de negócio são claras e sem ambiguidade?
- [ ] O escopo está delimitado (incluído vs. excluído)?

## 4. Fase 2: DESIGN (Projeto)

### O que produzir

A fase de Design responde: **"COMO vamos construir?"**

| Artefato              | Conteúdo                                          | Exemplo no AskWise             |
|-----------------------|---------------------------------------------------|--------------------------------|
| Arquitetura           | Componentes, camadas, fluxos, decisões técnicas   | `01-ARQUITETURA.md`            |
| Modelo de Dados       | Tabelas, schemas, relacionamentos, structs        | `02-MODELO-DADOS.md`           |
| API Design            | Endpoints, payloads, status codes, exemplos       | `03-API-DESIGN.md`             |

### Princípios de Design

1. **Separação de Responsabilidades**: Cada componente faz UMA coisa
2. **Interfaces bem definidas**: Contratos claros entre componentes
3. **Decisões justificadas**: Documentar POR QUÊ cada escolha foi feita
4. **Diagramas visuais**: ASCII art, fluxos, tabelas — tudo que facilite entendimento

### Checklist do Design

- [ ] A arquitetura está clara em diagrama?
- [ ] Cada componente tem responsabilidade definida?
- [ ] Os fluxos de dados estão mapeados?
- [ ] O modelo de dados cobre todos os requisitos?
- [ ] A API está definida com exemplos?
- [ ] As decisões técnicas estão justificadas?
- [ ] As interfaces/contratos estão definidos?

## 5. Fase 3: DEVELOPMENT (Desenvolvimento)

### O que produzir

A fase de Development responde: **"Vamos CONSTRUIR seguindo spec e design."**

| Artefato              | Conteúdo                                          |
|-----------------------|---------------------------------------------------|
| Código fonte          | Implementação seguindo a arquitetura definida      |
| Testes                | Unitários e de integração validando requisitos     |
| Documentação no código| Comentários explicando o "porquê", não o "o quê"  |
| README                | Como rodar, configurar, usar                       |
| Makefile              | Automatização de tarefas comuns                    |

### Ordem de Desenvolvimento Recomendada

Para o AskWise, a ordem sugerida é bottom-up (infraestrutura → negócio → interface):

```
Fase 1: Infraestrutura
├── internal/storage    (SQLite)
├── internal/vectorstore (Qdrant client)
├── internal/embedding   (OpenAI embeddings)
└── internal/llm         (OpenAI chat)

Fase 2: Lógica de Negócio
├── internal/document    (parsers de arquivos)
├── internal/chunker     (divisão de texto)
├── internal/retriever   (busca de contexto)
└── internal/rag         (orquestrador)

Fase 3: Interfaces
├── cmd/server           (API HTTP)
└── cmd/chat             (CLI terminal)
```

### Boas Práticas no Desenvolvimento

- **Implemente uma interface de cada vez**: Comece pelo storage, depois embedding, etc.
- **Teste isoladamente**: Cada componente deve funcionar independente
- **Commits frequentes**: Um commit por componente implementado
- **Código comentado**: Este é um projeto educativo, comente generosamente
- **Siga o Design**: Não invente coisas que não estão no design

## 6. Fluxo SDD na Prática

```
                    ┌─────────────┐
                    │  IDEIA      │
                    │  "Quero um  │
                    │  chatbot    │
                    │  de suporte"│
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │    SPEC     │  ← Estamos aqui no AskWise!
                    │             │
                    │ Requisitos  │  📄 01-VISAO-GERAL.md
                    │ Regras      │  📄 02-CENARIO-NEGOCIO.md
                    │ Cenário     │  📄 03-REQUISITOS.md
                    │             │  📄 04-REGRAS-NEGOCIO.md
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   DESIGN    │
                    │             │
                    │ Arquitetura │  📄 01-ARQUITETURA.md
                    │ Dados       │  📄 02-MODELO-DADOS.md
                    │ API         │  📄 03-API-DESIGN.md
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │ DEVELOPMENT │
                    │             │
                    │ Código      │  📦 internal/...
                    │ Testes      │  📦 cmd/...
                    │ Docs        │  📦 docker-compose.yml
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  ENTREGA    │
                    │             │
                    │  Sistema    │
                    │  funcional  │
                    └─────────────┘
```

## 7. SDD vs Outras Metodologias

| Aspecto              | SDD              | Agile/Scrum       | Waterfall         |
|----------------------|------------------|-------------------|-------------------|
| Documentação upfront | Sim (Spec+Design)| Mínima            | Extensiva         |
| Iterações            | Por fase         | Sprints curtos    | Uma grande fase   |
| Flexibilidade        | Média            | Alta              | Baixa             |
| Adequado para IA     | Excelente        | Razoável          | Razoável          |
| Complexidade         | Baixa            | Média             | Alta              |
| Ideal para           | PoCs, MVPs, times pequenos | Produtos em evolução | Sistemas críticos |

## 8. Dicas para Aplicar SDD

1. **Não pule fases**: A tentação de "ir direto pro código" é grande, mas gera retrabalho
2. **Spec não precisa ser perfeita**: 80% definido é suficiente para começar o Design
3. **Design evolui**: É normal ajustar o design durante o desenvolvimento
4. **Use Markdown**: Documentos simples, versionados no Git, legíveis por humanos e IAs
5. **Referência cruzada**: RF-01 no código referencia RF-01 na spec
6. **Mantenha atualizado**: Se o código mudar, atualize spec e design

## 9. Como SDD Funciona com Cursor/IA

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Você: "Implemente o internal/chunker seguindo           │
│         a spec (03-REQUISITOS.md, RN-06 a RN-09)        │
│         e o design (01-ARQUITETURA.md, seção 2.4)"       │
│                                                          │
│  IA: *lê os documentos referenciados*                    │
│      *entende chunk_size=500, overlap=50*                │
│      *vê a interface Chunker definida*                   │
│      *implementa exatamente o que foi especificado*      │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

Os documentos SDD servem como **instruções precisas** para a IA, resultando em
código muito mais alinhado com o que você realmente precisa.
