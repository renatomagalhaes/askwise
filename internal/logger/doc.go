// Package logger configura o logger estruturado JSON do AskWise usando
// log/slog da standard library.
//
// ADR-002: Logs estruturados em JSON via STDOUT/STDERR
// Spec dirigindo: RNF-04 (observabilidade)
//
// Todos os logs são emitidos em formato JSON. Logs de nível INFO e DEBUG
// vão para STDOUT; WARN e ERROR vão para STDERR. Isso segue o padrão
// 12-factor apps e facilita o consumo por ferramentas como `docker logs`,
// `jq` e stacks de observabilidade (ELK, Loki, etc.).
//
// Formato de saída:
//
//	{"time":"2025-01-15T10:30:00Z","level":"INFO","msg":"server started","component":"server","port":8484}
//
// Exemplo de uso:
//
//	log := logger.New("server")
//	log.Info("server started", "port", 8484)
//	log.Error("failed to connect", "error", err)
package logger
