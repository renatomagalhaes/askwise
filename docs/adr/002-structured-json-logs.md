# ADR-002: Logs Estruturados em JSON via STDOUT/STDERR

## Status

Aceita

## Contexto

O AskWise precisa de observabilidade para debug durante desenvolvimento e para
monitoramento em produção. Precisamos decidir o formato e destino dos logs.

Opções consideradas:

1. **Texto livre (fmt.Println)**: Simples, legível, mas não parseável
2. **Log estruturado JSON (STDOUT/STDERR)**: Parseável, padrão de containers
3. **Arquivo de log**: Tradicional, mas não se alinha com containers
4. **Biblioteca externa (zerolog, zap)**: Poderosas mas adicionam dependência

## Decisão

Adotamos **logs estruturados em JSON** enviados para **STDOUT** (info, debug) e
**STDERR** (warn, error, fatal). Usamos `log/slog` da standard library do Go (1.21+).

## Justificativa

- **Standard library**: `log/slog` é nativo do Go 1.21+, sem dependências externas
- **JSON parseável**: Ferramentas como `jq`, `docker logs`, ELK stack podem processar
- **STDOUT/STDERR**: Padrão 12-factor apps e containers Docker
- **Campos estruturados**: Permite filtrar por level, component, doc_id, etc.
- **Educativo**: Ensina boas práticas de observabilidade desde o início

## Formato

```json
{
  "time": "2025-01-15T10:30:00.000Z",
  "level": "INFO",
  "msg": "document uploaded",
  "component": "server",
  "doc_id": "a1b2c3d4",
  "file_name": "manual.pdf",
  "chunks": 42,
  "duration_ms": 1523
}
```

### Campos Padrão

| Campo        | Tipo   | Descrição                                     |
|--------------|--------|-----------------------------------------------|
| `time`       | string | Timestamp ISO 8601 / RFC 3339                 |
| `level`      | string | INFO, WARN, ERROR, DEBUG                      |
| `msg`        | string | Mensagem do log                               |
| `component`  | string | Componente que gerou o log (server, chat, rag)|
| `error`      | string | Mensagem de erro (apenas em ERROR)            |

### Níveis

| Nível | Destino | Uso                                              |
|-------|---------|--------------------------------------------------|
| DEBUG | STDOUT  | Detalhes internos (chunking, embedding sizes)    |
| INFO  | STDOUT  | Eventos normais (upload, query, response)        |
| WARN  | STDERR  | Situações inesperadas mas recuperáveis           |
| ERROR | STDERR  | Falhas que impactam funcionalidade               |

## Consequências

### Positivas
- `docker compose logs` mostra logs formatados
- `make logs | jq .` permite filtrar/formatar
- Compatível com qualquer stack de observabilidade
- Zero dependências externas

### Negativas
- JSON é menos legível que texto livre no terminal humano
- Mitigação: `make logs` pode usar `jq` para colorir e formatar
