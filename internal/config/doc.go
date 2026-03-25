// Package config carrega e valida as variáveis de ambiente necessárias
// para o funcionamento do AskWise.
//
// Spec dirigindo: RNF-03 (simplicidade), RNF-05 (segurança)
//
// Toda a configuração vem de variáveis de ambiente, seguindo o padrão
// 12-factor apps. Em ambiente Docker, as variáveis são injetadas via
// docker-compose.yml (env_file). Fora do Docker, o desenvolvedor pode
// usar um arquivo .env carregado manualmente.
//
// Exemplo de uso:
//
//	cfg, err := config.Load()
//	if err != nil {
//	    log.Fatal("falha ao carregar configuração", "error", err)
//	}
//	fmt.Println(cfg.ServerPort) // "8484"
package config
