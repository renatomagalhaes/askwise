# ============================================================================
# AskWise — Makefile
# ============================================================================
# ADR-001: Docker-first — todos os comandos rodam dentro de containers.
# Nenhuma instalação local de Go é necessária. Apenas Docker + Make.
#
# Uso:
#   make help       — Lista todos os comandos disponíveis
#   make up         — Sobe toda a infra (qdrant + app)
#   make chat       — Abre o chat interativo
#   make test       — Roda testes unitários
# ============================================================================

.PHONY: help up down restart build test test-integration test-coverage lint \
        chat logs health upload clean dev-shell mod-tidy

# Variáveis
COMPOSE       = docker compose
COMPOSE_RUN   = $(COMPOSE) run --rm
APP_SERVICE   = app
CHAT_SERVICE  = chat
API_URL       = http://localhost:8484/api/v1

# Cores para output
GREEN  = \033[0;32m
YELLOW = \033[0;33m
CYAN   = \033[0;36m
RESET  = \033[0m

# ============================================================================
# HELP
# ============================================================================

help: ## Mostra esta mensagem de ajuda
	@echo ""
	@echo "$(CYAN)AskWise$(RESET) — Comandos disponíveis:"
	@echo ""
	@echo "$(GREEN)Infraestrutura:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; /^(up|down|restart|build|clean|dev-shell)/ {printf "  $(YELLOW)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Desenvolvimento:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; /^(test|lint|mod)/ {printf "  $(YELLOW)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Aplicação:$(RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; /^(chat|logs|health|upload)/ {printf "  $(YELLOW)%-20s$(RESET) %s\n", $$1, $$2}'
	@echo ""

# ============================================================================
# INFRAESTRUTURA
# ============================================================================

up: ## Sobe toda a infra (qdrant + app)
	@echo "$(GREEN)Subindo AskWise...$(RESET)"
	$(COMPOSE) up -d
	@echo "$(GREEN)AskWise rodando!$(RESET)"
	@echo "  API:    http://localhost:8484"
	@echo "  Qdrant: http://localhost:6333"

down: ## Derruba todos os containers
	@echo "$(YELLOW)Parando AskWise...$(RESET)"
	$(COMPOSE) --profile chat down

restart: ## Reinicia todos os containers
	$(COMPOSE) --profile chat restart

build: ## Builda as imagens Docker (sem cache)
	$(COMPOSE) build --no-cache

clean: ## Remove containers, volumes e dados locais
	@echo "$(YELLOW)Limpando tudo...$(RESET)"
	$(COMPOSE) --profile chat down -v
	rm -rf data/

dev-shell: ## Abre um shell dentro do container de desenvolvimento
	$(COMPOSE_RUN) $(APP_SERVICE) sh

# ============================================================================
# DESENVOLVIMENTO
# ============================================================================

test: ## Roda testes unitários
	$(COMPOSE_RUN) $(APP_SERVICE) go test -v -count=1 -race ./...

test-integration: ## Roda testes de integração (precisa de Qdrant rodando)
	$(COMPOSE_RUN) $(APP_SERVICE) go test -v -count=1 -race -tags=integration ./...

test-coverage: ## Roda testes com cobertura
	$(COMPOSE_RUN) $(APP_SERVICE) go test -v -count=1 -race -coverprofile=coverage.out ./...
	$(COMPOSE_RUN) $(APP_SERVICE) go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Relatório de cobertura gerado: coverage.html$(RESET)"

lint: ## Roda linter (go vet)
	$(COMPOSE_RUN) $(APP_SERVICE) go vet ./...

mod-tidy: ## Organiza go.mod e go.sum
	$(COMPOSE_RUN) $(APP_SERVICE) go mod tidy

# ============================================================================
# APLICAÇÃO
# ============================================================================

chat: ## Abre o chat interativo no terminal
	@echo "$(GREEN)Iniciando AskWise Chat...$(RESET)"
	$(COMPOSE) run --rm --no-deps=false chat

logs: ## Mostra logs de todos os serviços (follow)
	$(COMPOSE) logs -f

logs-app: ## Mostra logs apenas da aplicação (follow)
	$(COMPOSE) logs -f $(APP_SERVICE)

health: ## Verifica saúde da API
	@curl -s $(API_URL)/health | jq . 2>/dev/null || echo "API não está respondendo. Execute 'make up' primeiro."

upload: ## Upload de arquivo. Uso: make upload FILE=caminho/do/arquivo.pdf
ifndef FILE
	@echo "$(YELLOW)Uso: make upload FILE=caminho/do/arquivo.pdf$(RESET)"
	@echo "Formatos aceitos: pdf, csv, txt, yaml, json, md"
else
	@echo "$(GREEN)Enviando $(FILE)...$(RESET)"
	@curl -s -X POST $(API_URL)/documents -F "file=@$(FILE)" | jq .
endif
