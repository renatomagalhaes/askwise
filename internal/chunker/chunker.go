package chunker

import (
	"github.com/renatomagalhaes/askwise/internal/document"
)

// Chunk representa um pedaço de texto de um documento, pronto para embedding.
// Spec: design/02-MODELO-DADOS.md §4 (Chunk)
//
// RN-08: Todo chunk deve conter document_id, chunk_index, file_name, file_type.
// O ID e DocumentID são preenchidos pelo orquestrador RAG (etapa 4),
// pois dependem do UUID do documento que é gerado no momento do upload.
type Chunk struct {
	// ID é o UUID único do chunk (preenchido pelo orquestrador).
	ID string

	// DocumentID é o UUID do documento pai (preenchido pelo orquestrador).
	DocumentID string

	// Index é a posição deste chunk dentro do documento (0-based).
	Index int

	// Text é o conteúdo textual do chunk.
	Text string

	// FileName é o nome do arquivo fonte (para citação de fontes, RN-14).
	FileName string

	// FileType é o tipo do arquivo fonte (ex: "pdf", "md").
	FileType string
}

// Chunker define a interface para divisão de documentos em chunks.
// Spec: design/01-ARQUITETURA.md §2.4
//
// A interface permite trocar a estratégia de chunking (ex: por token count,
// por semântica) sem alterar o restante do pipeline.
type Chunker interface {
	// Chunk divide um documento em pedaços menores.
	// Cada chunk contém Index, Text, FileName e FileType preenchidos.
	// ID e DocumentID ficam vazios (preenchidos pelo orquestrador).
	Chunk(doc *document.Document) ([]Chunk, error)
}
