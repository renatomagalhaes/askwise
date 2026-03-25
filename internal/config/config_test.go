package config

import (
	"os"
	"testing"
)

// TestLoadWithoutAPIKey verifica que Load retorna erro quando OPENAI_API_KEY
// não está definida — RNF-05 exige que a key nunca esteja hardcoded.
func TestLoadWithoutAPIKey(t *testing.T) {
	// Limpa a variável para garantir que o teste é isolado
	os.Unsetenv("OPENAI_API_KEY")

	_, err := Load()
	if err == nil {
		t.Error("Load() deveria retornar erro sem OPENAI_API_KEY, mas retornou nil")
	}
}

// TestLoadWithPlaceholderKey verifica que a key placeholder do .env.example
// também é rejeitada — evita que alguém rode sem configurar.
func TestLoadWithPlaceholderKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-your-api-key-here")

	_, err := Load()
	if err == nil {
		t.Error("Load() deveria rejeitar a key placeholder, mas aceitou")
	}
}

// TestLoadWithValidKey verifica que Load funciona quando uma API key válida
// é fornecida.
func TestLoadWithValidKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-key-1234567890")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() retornou erro inesperado: %v", err)
	}

	if cfg.OpenAIAPIKey != "sk-test-key-1234567890" {
		t.Errorf("OpenAIAPIKey = %q, esperava %q", cfg.OpenAIAPIKey, "sk-test-key-1234567890")
	}
}

// TestDefaults verifica que os valores padrão são aplicados corretamente
// quando as variáveis de ambiente não estão definidas.
func TestDefaults(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	// Limpa variáveis para forçar os defaults
	for _, key := range []string{
		"OPENAI_EMBEDDING_MODEL", "OPENAI_CHAT_MODEL",
		"QDRANT_HOST", "QDRANT_PORT", "QDRANT_COLLECTION",
		"SERVER_PORT", "MAX_FILE_SIZE",
		"RAG_CHUNK_SIZE", "RAG_CHUNK_OVERLAP", "RAG_TOP_K", "RAG_SCORE_THRESHOLD",
		"SQLITE_PATH",
	} {
		os.Unsetenv(key)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() retornou erro: %v", err)
	}

	// Table-driven tests para verificar cada default
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"OpenAIEmbeddingModel", cfg.OpenAIEmbeddingModel, "text-embedding-3-small"},
		{"OpenAIChatModel", cfg.OpenAIChatModel, "gpt-4o-mini"},
		{"QdrantHost", cfg.QdrantHost, "localhost"},
		{"QdrantPort", cfg.QdrantPort, 6333},
		{"QdrantCollection", cfg.QdrantCollection, "askwise"},
		{"ServerPort", cfg.ServerPort, 8484},
		{"MaxFileSize", cfg.MaxFileSize, int64(10485760)},
		{"RAGChunkSize", cfg.RAGChunkSize, 500},
		{"RAGChunkOverlap", cfg.RAGChunkOverlap, 50},
		{"RAGTopK", cfg.RAGTopK, 5},
		{"RAGScoreThreshold", cfg.RAGScoreThreshold, 0.5},
		{"SQLitePath", cfg.SQLitePath, "./data/askwise.db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, esperava %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

// TestCustomValues verifica que variáveis de ambiente customizadas
// sobrescrevem os valores padrão.
func TestCustomValues(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-custom")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("RAG_TOP_K", "10")
	t.Setenv("RAG_SCORE_THRESHOLD", "0.7")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() retornou erro: %v", err)
	}

	if cfg.ServerPort != 9090 {
		t.Errorf("ServerPort = %d, esperava 9090", cfg.ServerPort)
	}
	if cfg.RAGTopK != 10 {
		t.Errorf("RAGTopK = %d, esperava 10", cfg.RAGTopK)
	}
	if cfg.RAGScoreThreshold != 0.7 {
		t.Errorf("RAGScoreThreshold = %f, esperava 0.7", cfg.RAGScoreThreshold)
	}
}

// TestHelperMethods verifica os métodos utilitários do Config.
func TestHelperMethods(t *testing.T) {
	cfg := &Config{
		QdrantHost:  "qdrant",
		QdrantPort:  6333,
		ServerPort:  8484,
		OpenAIAPIKey: "sk-test",
	}

	if got := cfg.QdrantAddr(); got != "qdrant:6333" {
		t.Errorf("QdrantAddr() = %q, esperava %q", got, "qdrant:6333")
	}

	if got := cfg.ServerAddr(); got != ":8484" {
		t.Errorf("ServerAddr() = %q, esperava %q", got, ":8484")
	}

	if !cfg.OpenAIConfigured() {
		t.Error("OpenAIConfigured() = false, esperava true")
	}

	cfg.OpenAIAPIKey = ""
	if cfg.OpenAIConfigured() {
		t.Error("OpenAIConfigured() = true sem key, esperava false")
	}
}
