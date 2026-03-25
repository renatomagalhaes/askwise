# Cenário de Negócio — TechSupport Ltda.

## 1. A Empresa Fictícia

**TechSupport Ltda.** é uma empresa de suporte técnico que atende clientes de software
empresarial. Ela opera há 12 anos e tem:

- **45 atendentes** divididos em 3 níveis (N1, N2, N3)
- **~200 clientes** ativos com contratos de suporte
- **~500 tickets/dia** de atendimento
- **12 produtos** diferentes que precisam de suporte

## 2. O Problema Real

### Conhecimento Espalhado

O conhecimento da empresa está distribuído em:

```
📁 SharePoint (312 documentos)
├── 📄 Manuais de produto (PDF) ........... 89 arquivos
├── 📄 Runbooks de operação (MD) .......... 67 arquivos
├── 📄 FAQs e troubleshooting (TXT) ....... 45 arquivos
├── 📄 Configurações padrão (YAML/JSON) ... 38 arquivos
├── 📄 Relatórios de incidentes (CSV) ..... 41 arquivos
└── 📄 Procedimentos internos (PDF) ....... 32 arquivos
```

### Cenários do Dia-a-Dia

**Cenário 1 — O Atendente Novo:**
> João entrou na empresa há 2 semanas. Um cliente liga perguntando sobre o erro
> "CONNECTION_TIMEOUT_ERR_5032" no produto XPro. João não sabe a resposta, precisa
> escalar para N2. O atendente N2 já viu esse erro 50 vezes e resolve em 2 minutos.
> Com AskWise, João perguntaria ao chatbot e teria a resposta imediata.

**Cenário 2 — O Conhecimento Perdido:**
> Maria era a especialista em integrações do produto DataSync. Ela saiu da empresa
> levando consigo 8 anos de conhecimento. Agora ninguém sabe resolver problemas
> complexos de sincronização. Com AskWise, o conhecimento de Maria (documentado em
> runbooks e notas) estaria preservado e acessível.

**Cenário 3 — A Busca Impossível:**
> Carlos precisa encontrar o procedimento para resetar o cache do produto CloudAPI.
> Ele sabe que existe um documento, mas não lembra o nome nem onde está salvo.
> Busca por "cache" no SharePoint e encontra 47 resultados. Com AskWise, ele
> perguntaria "como resetar o cache do CloudAPI?" e teria a resposta direta.

**Cenário 4 — A Resposta Inconsistente:**
> O cliente Acme Corp liga 3 vezes sobre o mesmo problema. Cada atendente dá uma
> resposta diferente. O cliente fica frustrado. Com AskWise, todos receberiam a
> mesma resposta baseada na documentação oficial.

## 3. Personas

### Ana — Gestora de Conhecimento
- **Papel**: Responsável por manter a base de documentos atualizada
- **Uso do AskWise**: Faz upload de novos documentos via API
- **Dor**: Passa horas organizando documentos que ninguém encontra
- **Ganho**: Basta fazer upload, o AskWise indexa e disponibiliza automaticamente

### Pedro — Atendente N1 (Júnior)
- **Papel**: Primeiro contato com o cliente, resolve problemas simples
- **Uso do AskWise**: Pergunta ao chatbot antes de escalar para N2
- **Dor**: Precisa escalar 60% dos tickets porque não tem conhecimento
- **Ganho**: Resolve 80% dos tickets sozinho com ajuda do chatbot

### Carla — Atendente N3 (Sênior)
- **Papel**: Resolve problemas complexos, cria documentação
- **Uso do AskWise**: Valida respostas do chatbot, contribui com novos documentos
- **Dor**: Interrompida constantemente por perguntas que já documentou
- **Ganho**: Menos interrupções, foca em problemas realmente complexos

### Ricardo — Gerente de Suporte
- **Papel**: Gerencia o time, monitora métricas de atendimento
- **Uso do AskWise**: Avalia impacto na redução de tempo de atendimento
- **Dor**: TMR (Tempo Médio de Resolução) alto, muitas escalações
- **Ganho**: TMR reduzido em 40%, escalações reduzidas em 50%

## 4. Jornada do Usuário

### Jornada 1: Upload de Documentos (Ana)

```
Ana (Gestora)          API AskWise             Sistema
     │                      │                      │
     │  POST /documents     │                      │
     │  + arquivo PDF       │                      │
     │─────────────────────▶│                      │
     │                      │  Valida formato      │
     │                      │─────────────────────▶│
     │                      │                      │
     │                      │  Extrai texto         │
     │                      │─────────────────────▶│
     │                      │                      │
     │                      │  Divide em chunks     │
     │                      │─────────────────────▶│
     │                      │                      │
     │                      │  Gera embeddings      │
     │                      │─────────────────────▶│
     │                      │                      │
     │                      │  Armazena vetores     │
     │                      │─────────────────────▶│
     │                      │                      │
     │  201 Created         │                      │
     │  {doc_id, chunks: 42}│                      │
     │◀─────────────────────│                      │
```

### Jornada 2: Chat com o Chatbot (Pedro)

```
Pedro (N1)             CLI AskWise              Sistema
     │                      │                      │
     │  "Como resolver o    │                      │
     │   erro 5032?"        │                      │
     │─────────────────────▶│                      │
     │                      │  Gera embedding       │
     │                      │  da pergunta          │
     │                      │─────────────────────▶│
     │                      │                      │
     │                      │  Busca chunks         │
     │                      │  similares (top 5)    │
     │                      │─────────────────────▶│
     │                      │                      │
     │                      │  Monta prompt com     │
     │                      │  contexto + pergunta  │
     │                      │─────────────────────▶│
     │                      │                      │
     │                      │  LLM gera resposta    │
     │                      │─────────────────────▶│
     │                      │                      │
     │  "O erro 5032 é...   │                      │
     │   Para resolver:     │                      │
     │   1. Verifique...    │                      │
     │   Fonte: runbook-    │                      │
     │   cloudapi.md"       │                      │
     │◀─────────────────────│                      │
```

## 5. Resultados Esperados

| Métrica                  | Antes do AskWise | Depois do AskWise | Melhoria |
|--------------------------|------------------|--------------------|----------|
| Tempo Médio de Resolução | 25 min           | 10 min             | -60%     |
| Taxa de Escalação N1→N2  | 60%              | 25%                | -58%     |
| Onboarding (produtivo)   | 3 meses          | 3 semanas          | -75%     |
| Satisfação do Cliente    | 3.2/5            | 4.5/5              | +40%     |
| Consistência Respostas   | 45%              | 95%                | +111%    |

## 6. Documentos de Exemplo

Para testar o AskWise, usaremos documentos fictícios da TechSupport Ltda:

| Arquivo                           | Tipo | Conteúdo                                    |
|-----------------------------------|------|---------------------------------------------|
| `manual-cloudapi-v3.pdf`          | PDF  | Manual do produto CloudAPI versão 3         |
| `runbook-datasync.md`             | MD   | Procedimentos operacionais do DataSync      |
| `faq-erros-comuns.txt`            | TXT  | FAQ com erros mais frequentes e soluções    |
| `config-padrao-xpro.yaml`        | YAML | Configuração padrão recomendada do XPro     |
| `endpoints-integracao.json`       | JSON | Endpoints de integração entre produtos      |
| `incidentes-2024.csv`            | CSV  | Histórico de incidentes com resoluções      |
