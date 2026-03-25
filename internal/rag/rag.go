package rag

import (
	"github.com/renatomagalhaes/askwise/internal/chunker"
	"github.com/renatomagalhaes/askwise/internal/document"
	"github.com/renatomagalhaes/askwise/internal/embedding"
	"github.com/renatomagalhaes/askwise/internal/llm"
	"github.com/renatomagalhaes/askwise/internal/retriever"
	"github.com/renatomagalhaes/askwise/internal/storage"
	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

// RAG é o orquestrador central do pipeline.
// Spec: design/01-ARQUITETURA.md §2.9
//
// Conecta todos os componentes via interfaces injetadas no construtor.
type RAG struct {
	parser     document.Parser
	chunker    chunker.Chunker
	embedder   embedding.Embedder
	vectorStore vectorstore.VectorStore
	retriever  retriever.Retriever
	llm        llm.LLM
	storage    storage.Storage

	// collection é o nome da collection no VectorStore.
	collection string
}

// Config agrupa os parâmetros de configuração do RAG.
type Config struct {
	Collection string
}

// NewRAG cria uma nova instância do orquestrador com todos os componentes injetados.
func NewRAG(
	parser document.Parser,
	chk chunker.Chunker,
	embedder embedding.Embedder,
	vs vectorstore.VectorStore,
	ret retriever.Retriever,
	l llm.LLM,
	store storage.Storage,
	cfg Config,
) *RAG {
	return &RAG{
		parser:      parser,
		chunker:     chk,
		embedder:    embedder,
		vectorStore: vs,
		retriever:   ret,
		llm:         l,
		storage:     store,
		collection:  cfg.Collection,
	}
}

// Source representa a origem de um trecho usado na resposta.
// RN-14: Toda resposta deve incluir referência aos documentos utilizados.
type Source struct {
	FileName string  `json:"file_name"`
	FileType string  `json:"file_type"`
	Index    int     `json:"chunk_index"`
	Score    float32 `json:"score"`
}

// QueryResponse é o resultado de uma consulta ao pipeline RAG.
type QueryResponse struct {
	// Answer é a resposta gerada pela LLM.
	Answer string `json:"answer"`

	// Sources são os documentos/chunks usados para gerar a resposta.
	Sources []Source `json:"sources"`
}
