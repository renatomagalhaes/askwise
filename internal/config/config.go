package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config armazena todas as configurações do AskWise carregadas das variáveis
// de ambiente. Cada campo corresponde a uma variável definida em .env.example.
type Config struct {
	// --- OpenAI API ---

	// OpenAIAPIKey é a chave de autenticação da API da OpenAI.
	// Variável: OPENAI_API_KEY (obrigatória)
	OpenAIAPIKey string

	// OpenAIEmbeddingModel é o modelo usado para gerar embeddings vetoriais.
	// Variável: OPENAI_EMBEDDING_MODEL (padrão: "text-embedding-3-small")
	OpenAIEmbeddingModel string

	// OpenAIChatModel é o modelo usado para gerar respostas do chat.
	// Variável: OPENAI_CHAT_MODEL (padrão: "gpt-4o-mini")
	OpenAIChatModel string

	// --- Qdrant ---

	// QdrantHost é o endereço do servidor Qdrant.
	// Variável: QDRANT_HOST (padrão: "localhost")
	QdrantHost string

	// QdrantPort é a porta REST do Qdrant.
	// Variável: QDRANT_PORT (padrão: 6333)
	QdrantPort int

	// QdrantCollection é o nome da collection no Qdrant onde os vetores são armazenados.
	// Variável: QDRANT_COLLECTION (padrão: "askwise")
	QdrantCollection string

	// --- Server HTTP ---

	// ServerPort é a porta onde a API HTTP escuta.
	// Variável: SERVER_PORT (padrão: 8484)
	ServerPort int

	// MaxFileSize é o tamanho máximo de arquivo aceito no upload, em bytes.
	// Variável: MAX_FILE_SIZE (padrão: 10485760 = 10MB)
	// RN-02: Limite de tamanho
	MaxFileSize int64

	// --- RAG Pipeline ---

	// RAGChunkSize é o tamanho alvo de cada chunk em tokens.
	// Variável: RAG_CHUNK_SIZE (padrão: 500)
	// RN-06: Tamanho de chunk
	RAGChunkSize int

	// RAGChunkOverlap é o número de tokens de sobreposição entre chunks consecutivos.
	// Variável: RAG_CHUNK_OVERLAP (padrão: 50)
	// RN-07: Overlap entre chunks
	RAGChunkOverlap int

	// RAGTopK é a quantidade de chunks retornados na busca vetorial.
	// Variável: RAG_TOP_K (padrão: 5)
	// RN-10: Número de resultados
	RAGTopK int

	// RAGScoreThreshold é o score mínimo de similaridade para um chunk ser considerado relevante.
	// Variável: RAG_SCORE_THRESHOLD (padrão: 0.5)
	// RN-11: Score mínimo de relevância
	RAGScoreThreshold float64

	// --- SQLite ---

	// SQLitePath é o caminho para o arquivo do banco de dados SQLite.
	// Variável: SQLITE_PATH (padrão: "./data/askwise.db")
	SQLitePath string
}

// Load carrega a configuração a partir das variáveis de ambiente.
// Retorna erro se alguma variável obrigatória estiver ausente.
//
// As variáveis são injetadas pelo Docker Compose via env_file (.env).
// Cada variável tem um valor padrão definido, exceto OPENAI_API_KEY que é obrigatória.
func Load() (*Config, error) {
	cfg := &Config{
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAIEmbeddingModel: getEnvOrDefault("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		OpenAIChatModel:      getEnvOrDefault("OPENAI_CHAT_MODEL", "gpt-4o-mini"),
		QdrantHost:           getEnvOrDefault("QDRANT_HOST", "localhost"),
		QdrantPort:           getEnvIntOrDefault("QDRANT_PORT", 6333),
		QdrantCollection:     getEnvOrDefault("QDRANT_COLLECTION", "askwise"),
		ServerPort:           getEnvIntOrDefault("SERVER_PORT", 8484),
		MaxFileSize:          getEnvInt64OrDefault("MAX_FILE_SIZE", 10485760),
		RAGChunkSize:         getEnvIntOrDefault("RAG_CHUNK_SIZE", 500),
		RAGChunkOverlap:      getEnvIntOrDefault("RAG_CHUNK_OVERLAP", 50),
		RAGTopK:              getEnvIntOrDefault("RAG_TOP_K", 5),
		RAGScoreThreshold:    getEnvFloatOrDefault("RAG_SCORE_THRESHOLD", 0.5),
		SQLitePath:           getEnvOrDefault("SQLITE_PATH", "./data/askwise.db"),
	}

	// RNF-05: A API key nunca deve estar hardcoded. Validamos aqui que foi
	// fornecida via variável de ambiente. Sem ela, embeddings e chat não funcionam.
	if cfg.OpenAIAPIKey == "" || cfg.OpenAIAPIKey == "sk-your-api-key-here" {
		return cfg, fmt.Errorf("OPENAI_API_KEY não configurada: defina no arquivo .env")
	}

	return cfg, nil
}

// LoadOrWarn carrega a configuração sem falhar se OPENAI_API_KEY estiver ausente.
// Útil para o health check e para etapas que não precisam da OpenAI (ex: upload).
func LoadOrWarn() *Config {
	cfg, _ := Load()
	if cfg == nil {
		cfg = &Config{
			OpenAIEmbeddingModel: "text-embedding-3-small",
			OpenAIChatModel:      "gpt-4o-mini",
			QdrantHost:           getEnvOrDefault("QDRANT_HOST", "localhost"),
			QdrantPort:           getEnvIntOrDefault("QDRANT_PORT", 6333),
			QdrantCollection:     getEnvOrDefault("QDRANT_COLLECTION", "askwise"),
			ServerPort:           getEnvIntOrDefault("SERVER_PORT", 8484),
			MaxFileSize:          getEnvInt64OrDefault("MAX_FILE_SIZE", 10485760),
			RAGChunkSize:         getEnvIntOrDefault("RAG_CHUNK_SIZE", 500),
			RAGChunkOverlap:      getEnvIntOrDefault("RAG_CHUNK_OVERLAP", 50),
			RAGTopK:              getEnvIntOrDefault("RAG_TOP_K", 5),
			RAGScoreThreshold:    getEnvFloatOrDefault("RAG_SCORE_THRESHOLD", 0.5),
			SQLitePath:           getEnvOrDefault("SQLITE_PATH", "./data/askwise.db"),
		}
	}
	return cfg
}

// QdrantAddr retorna o endereço completo do Qdrant (host:port).
func (c *Config) QdrantAddr() string {
	return fmt.Sprintf("%s:%d", c.QdrantHost, c.QdrantPort)
}

// ServerAddr retorna o endereço de escuta do servidor HTTP (:port).
func (c *Config) ServerAddr() string {
	return fmt.Sprintf(":%d", c.ServerPort)
}

// OpenAIConfigured retorna true se a API key da OpenAI está configurada.
func (c *Config) OpenAIConfigured() bool {
	return c.OpenAIAPIKey != "" && c.OpenAIAPIKey != "sk-your-api-key-here"
}

// --- Helpers para leitura de variáveis de ambiente com defaults ---

// getEnvOrDefault retorna o valor da variável de ambiente ou o default fornecido.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvIntOrDefault retorna o valor inteiro da variável de ambiente ou o default.
func getEnvIntOrDefault(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return parsed
}

// getEnvInt64OrDefault retorna o valor int64 da variável de ambiente ou o default.
func getEnvInt64OrDefault(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return parsed
}

// getEnvFloatOrDefault retorna o valor float64 da variável de ambiente ou o default.
func getEnvFloatOrDefault(key string, defaultVal float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultVal
	}
	return parsed
}
