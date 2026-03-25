# ADR-001: Docker-First — Sem Go Instalado Localmente

## Status

Aceita

## Contexto

O AskWise é uma PoC educativa em Go que depende de serviços externos (Qdrant, OpenAI).
Precisamos decidir como o desenvolvedor compila, testa e executa o projeto.

Opções consideradas:

1. **Go local + Docker para serviços**: Desenvolvedor instala Go na máquina, Docker apenas para Qdrant
2. **Docker-first (tudo no container)**: Build, test, run — tudo via Docker, sem Go local
3. **DevContainer**: VSCode Dev Container com Go pré-instalado

## Decisão

Adotamos **Docker-first**: todo o tooling (compilação, testes, linting, execução) roda
dentro de containers Docker. O desenvolvedor só precisa de Docker e Make instalados.

## Justificativa

- **Zero setup**: `git clone` + `make up` = projeto rodando
- **Consistência**: Mesmo ambiente em qualquer máquina (macOS, Linux, Windows/WSL)
- **Sem conflito de versões**: Go version, CGO, dependências de sistema — tudo isolado
- **CI/CD aligned**: O mesmo Dockerfile usado em dev é usado em CI
- **Educativo**: Aprender Docker multi-stage builds é um objetivo de aprendizado válido

## Implementação

- `Dockerfile` multi-stage: stage `builder` compila, stage `runtime` executa
- `docker-compose.yml` orquestra `app` (API), `chat` (CLI) e `qdrant`
- `Makefile` expõe todos os comandos via targets que chamam `docker compose`
- Nenhum target do Makefile exige Go instalado localmente

## Consequências

### Positivas
- Onboarding em 1 comando
- Ambiente reproduzível
- Sem "funciona na minha máquina"

### Negativas
- Build mais lento que Go local (overhead do Docker layer)
- IDE (autocomplete, go-to-definition) pode precisar de Go local ou gopls remoto
- Debug mais complexo (precisa de `dlv` dentro do container ou remote debug)

### Mitigação
- Docker layer cache minimiza rebuilds
- Para IDE features, desenvolvedor pode opcionalmente instalar Go local
  apenas para tooling de editor (gopls), sem usá-lo para build/test/run
