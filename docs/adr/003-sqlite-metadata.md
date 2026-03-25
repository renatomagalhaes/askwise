# ADR-003: SQLite para Armazenamento de Metadados

## Status

Aceita

## Contexto

O AskWise precisa armazenar metadados dos documentos enviados (nome, tipo, tamanho,
status, timestamps). Os embeddings vetoriais ficam no Qdrant. Precisamos de um banco
para os metadados.

Opções consideradas:

| Opção         | Prós                                    | Contras                              |
|---------------|-----------------------------------------|--------------------------------------|
| SQLite        | Zero config, arquivo local, pure Go lib | Single-writer, não escala horizontal |
| PostgreSQL    | Robusto, pgvector possível              | Precisa de servidor, overkill p/ PoC |
| MySQL         | Popular, bom tooling                    | Precisa de servidor, overkill p/ PoC |
| MongoDB       | Schemaless, flexível                    | Precisa de servidor, outra linguagem |
| BoltDB/BBolt  | Embedded Go, key-value                  | Sem SQL, queries limitadas           |

## Decisão

Adotamos **SQLite** via `modernc.org/sqlite` (implementação pure Go, sem CGO).

## Justificativa

- **Zero configuração**: É um arquivo local, não precisa de servidor
- **Pure Go**: `modernc.org/sqlite` não precisa de CGO, compila em qualquer plataforma
- **SQL completo**: Queries, indexes, transactions — tudo disponível
- **Perfeito para PoC**: Volume de dados pequeno, single-user
- **Inspecionável**: Pode abrir o `.db` com qualquer client SQLite para debug
- **Docker-friendly**: O arquivo vive em volume montado

## Consequências

### Positivas
- Setup instantâneo (auto-create na primeira execução)
- Sem dependência de serviço externo para metadados
- Backup trivial (copiar o arquivo)

### Negativas
- Não escala horizontal (aceitável para PoC)
- Single-writer lock (aceitável para single-user)
- Se migrar para produção, precisaria trocar para PostgreSQL (interfaces facilitam)
