package chunker

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/renatomagalhaes/askwise/internal/document"
)

// defaultSeparators define a hierarquia de separadores para recursive splitting.
// Tenta dividir pelo separador mais significativo primeiro:
//   - "\n\n": parágrafos (preserva blocos de ideia)
//   - "\n": linhas (preserva estrutura)
//   - ". ": frases (preserva sentido)
//   - " ": palavras (último recurso)
var defaultSeparators = []string{"\n\n", "\n", ". ", " "}

// RecursiveChunker divide texto usando recursive character text splitting.
// RF-03: Chunks de ~chunkSize com overlap de ~overlapSize.
// RN-06: Cada chunk entre 100 e 500 tokens.
// RN-07: Overlap de ~50 tokens entre chunks consecutivos.
type RecursiveChunker struct {
	// ChunkSize é o tamanho alvo de cada chunk em caracteres.
	// Valor padrão: 500 (configurável via RAG_CHUNK_SIZE).
	ChunkSize int

	// OverlapSize é o número de caracteres de sobreposição entre chunks consecutivos.
	// Valor padrão: 50 (configurável via RAG_CHUNK_OVERLAP).
	OverlapSize int

	// Separators é a lista de separadores em ordem de prioridade.
	// Se não fornecido, usa defaultSeparators.
	Separators []string
}

// NewRecursiveChunker cria um RecursiveChunker com os parâmetros fornecidos.
func NewRecursiveChunker(chunkSize, overlapSize int) *RecursiveChunker {
	if chunkSize <= 0 {
		chunkSize = 500
	}
	if overlapSize < 0 {
		overlapSize = 0
	}
	if overlapSize >= chunkSize {
		overlapSize = chunkSize / 10
	}

	return &RecursiveChunker{
		ChunkSize:   chunkSize,
		OverlapSize: overlapSize,
		Separators:  defaultSeparators,
	}
}

// Chunk divide um Document em chunks com metadados.
// Preenche Index, Text, FileName e FileType em cada chunk.
func (rc *RecursiveChunker) Chunk(doc *document.Document) ([]Chunk, error) {
	if doc == nil {
		return nil, fmt.Errorf("documento é nil")
	}

	content := strings.TrimSpace(doc.Content)
	if content == "" {
		return nil, fmt.Errorf("documento %s não tem conteúdo para chunking", doc.Name)
	}

	// Divide o texto em partes usando recursive splitting.
	parts := rc.recursiveSplit(content, rc.Separators)

	// Aplica overlap entre chunks consecutivos.
	parts = rc.addOverlap(parts)

	// Monta os chunks com metadados.
	chunks := make([]Chunk, len(parts))
	for i, text := range parts {
		chunks[i] = Chunk{
			Index:    i,
			Text:     text,
			FileName: doc.Name,
			FileType: doc.FileType,
		}
	}

	slog.Info("document chunked",
		"component", "chunker",
		"filename", doc.Name,
		"chunks", len(chunks),
		"content_length", len(content),
		"chunk_size", rc.ChunkSize,
		"overlap", rc.OverlapSize,
	)

	return chunks, nil
}

// recursiveSplit divide o texto usando a hierarquia de separadores.
// Tenta o primeiro separador; se os pedaços resultantes forem maiores que
// chunkSize, aplica recursivamente o próximo separador.
func (rc *RecursiveChunker) recursiveSplit(text string, separators []string) []string {
	if len(text) <= rc.ChunkSize {
		return []string{text}
	}

	if len(separators) == 0 {
		// Sem mais separadores: corta no tamanho máximo.
		return rc.hardSplit(text)
	}

	sep := separators[0]
	remainingSeps := separators[1:]

	parts := strings.Split(text, sep)

	// Mescla partes pequenas em chunks do tamanho alvo.
	merged := rc.mergeParts(parts, sep)

	// Recursivamente divide chunks que ainda estão grandes demais.
	var result []string
	for _, chunk := range merged {
		if len(chunk) > rc.ChunkSize && len(remainingSeps) > 0 {
			subParts := rc.recursiveSplit(chunk, remainingSeps)
			result = append(result, subParts...)
		} else if len(chunk) > rc.ChunkSize {
			subParts := rc.hardSplit(chunk)
			result = append(result, subParts...)
		} else {
			result = append(result, chunk)
		}
	}

	return result
}

// mergeParts combina partes pequenas em chunks que se aproximam do chunkSize.
func (rc *RecursiveChunker) mergeParts(parts []string, sep string) []string {
	var merged []string
	var current strings.Builder

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Se adicionar esta parte ultrapassa o tamanho, fecha o chunk atual.
		newLen := current.Len() + len(sep) + len(part)
		if current.Len() > 0 && newLen > rc.ChunkSize {
			merged = append(merged, strings.TrimSpace(current.String()))
			current.Reset()
		}

		if current.Len() > 0 {
			current.WriteString(sep)
		}
		current.WriteString(part)
	}

	if current.Len() > 0 {
		merged = append(merged, strings.TrimSpace(current.String()))
	}

	return merged
}

// hardSplit divide o texto em pedaços de exatamente chunkSize quando nenhum
// separador funciona. Tenta quebrar na última palavra antes do limite.
func (rc *RecursiveChunker) hardSplit(text string) []string {
	var parts []string

	for len(text) > rc.ChunkSize {
		splitAt := rc.ChunkSize

		// Tenta quebrar no último espaço antes do limite para não cortar palavras.
		lastSpace := strings.LastIndex(text[:splitAt], " ")
		if lastSpace > rc.ChunkSize/2 {
			splitAt = lastSpace
		}

		parts = append(parts, strings.TrimSpace(text[:splitAt]))
		text = strings.TrimSpace(text[splitAt:])
	}

	if len(text) > 0 {
		parts = append(parts, strings.TrimSpace(text))
	}

	return parts
}

// addOverlap adiciona sobreposição entre chunks consecutivos.
// RN-07: O final do chunk anterior é prepended ao início do próximo chunk.
func (rc *RecursiveChunker) addOverlap(chunks []string) []string {
	if len(chunks) <= 1 || rc.OverlapSize <= 0 {
		return chunks
	}

	result := make([]string, len(chunks))
	result[0] = chunks[0]

	for i := 1; i < len(chunks); i++ {
		overlap := rc.getOverlapText(chunks[i-1])
		if overlap != "" {
			result[i] = overlap + " " + chunks[i]
		} else {
			result[i] = chunks[i]
		}
	}

	return result
}

// getOverlapText extrai os últimos ~overlapSize caracteres do texto,
// quebrando na última palavra completa.
func (rc *RecursiveChunker) getOverlapText(text string) string {
	if len(text) <= rc.OverlapSize {
		return text
	}

	// Pega os últimos overlapSize caracteres.
	tail := text[len(text)-rc.OverlapSize:]

	// Ajusta para começar na primeira palavra completa (após o primeiro espaço).
	firstSpace := strings.Index(tail, " ")
	if firstSpace >= 0 && firstSpace < len(tail)-1 {
		tail = tail[firstSpace+1:]
	}

	return strings.TrimSpace(tail)
}
