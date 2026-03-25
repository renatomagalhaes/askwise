// Package storage gerencia os metadados de documentos no SQLite.
//
// ADR-003: SQLite foi escolhido por ser zero-config, pure Go e perfeito para PoC.
// Spec dirigindo: RF-08 (listagem), RF-09 (remoção), RN-21 (integridade referencial)
//
// Este pacote NÃO armazena o conteúdo dos documentos nem os embeddings —
// apenas metadados como nome, tipo, tamanho, status e timestamps.
// Os embeddings ficam no Qdrant (internal/vectorstore).
//
// A tabela é criada automaticamente na primeira execução (auto-migrate).
package storage
