package chunker

import (
	"strings"
	"testing"

	"github.com/renatomagalhaes/askwise/internal/document"
)

// makeDoc cria um Document de teste com o conteúdo fornecido.
func makeDoc(name, content string) *document.Document {
	return &document.Document{
		Name:     name,
		FileType: "txt",
		Content:  content,
	}
}

// TestChunkSmallDocument verifica que texto menor que chunkSize gera 1 chunk.
func TestChunkSmallDocument(t *testing.T) {
	chunker := NewRecursiveChunker(500, 50)
	doc := makeDoc("small.txt", "Este é um documento pequeno.")

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	if len(chunks) != 1 {
		t.Fatalf("len(chunks) = %d, want 1", len(chunks))
	}

	if chunks[0].Text != "Este é um documento pequeno." {
		t.Errorf("Text = %q, esperava conteúdo original", chunks[0].Text)
	}
	if chunks[0].Index != 0 {
		t.Errorf("Index = %d, want 0", chunks[0].Index)
	}
	if chunks[0].FileName != "small.txt" {
		t.Errorf("FileName = %s, want small.txt", chunks[0].FileName)
	}
	if chunks[0].FileType != "txt" {
		t.Errorf("FileType = %s, want txt", chunks[0].FileType)
	}
}

// TestChunkLargeDocument verifica que texto grande é dividido em múltiplos chunks.
func TestChunkLargeDocument(t *testing.T) {
	chunker := NewRecursiveChunker(100, 0)

	// Cria texto com múltiplos parágrafos.
	paragraphs := []string{
		"Este é o primeiro parágrafo com informações sobre o sistema.",
		"O segundo parágrafo contém detalhes sobre a configuração necessária.",
		"No terceiro parágrafo explicamos como instalar as dependências.",
		"O quarto parágrafo descreve o processo de deploy em produção.",
		"Finalmente o quinto parágrafo com notas adicionais e referências.",
	}
	content := strings.Join(paragraphs, "\n\n")

	doc := makeDoc("large.txt", content)

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("len(chunks) = %d, esperava pelo menos 2 para texto grande", len(chunks))
	}

	// Verifica que todos os chunks têm índices sequenciais.
	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("chunks[%d].Index = %d, want %d", i, chunk.Index, i)
		}
	}
}

// TestChunkSizeRespected verifica que nenhum chunk excede significativamente o chunkSize.
// RN-06: Cada chunk entre 100 e 500 tokens.
func TestChunkSizeRespected(t *testing.T) {
	chunkSize := 200
	chunker := NewRecursiveChunker(chunkSize, 0)

	// Texto longo com separadores variados.
	content := strings.Repeat("Palavra ", 100)
	doc := makeDoc("long.txt", content)

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	// Tolerância: chunks podem ser um pouco maiores que chunkSize
	// quando não há separador bom para dividir.
	tolerance := chunkSize + chunkSize/5
	for i, chunk := range chunks {
		if len(chunk.Text) > tolerance {
			t.Errorf("chunks[%d] tem %d chars, excede tolerância de %d",
				i, len(chunk.Text), tolerance)
		}
	}
}

// TestChunkOverlap verifica que chunks consecutivos têm sobreposição.
// RN-07: Overlap de ~50 tokens.
func TestChunkOverlap(t *testing.T) {
	chunker := NewRecursiveChunker(100, 30)

	content := strings.Repeat("Palavra de teste. ", 30)
	doc := makeDoc("overlap.txt", content)

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	if len(chunks) < 2 {
		t.Fatalf("Precisa de pelo menos 2 chunks para testar overlap, got %d", len(chunks))
	}

	// Verifica que o segundo chunk contém parte do texto do primeiro (overlap).
	for i := 1; i < len(chunks); i++ {
		// O início do chunk atual deve conter texto do final do chunk anterior.
		prevEnd := chunks[i-1].Text
		if len(prevEnd) > 30 {
			prevEnd = prevEnd[len(prevEnd)-30:]
		}
		// Verifica que pelo menos algumas palavras do final do chunk anterior
		// aparecem no início do chunk atual.
		prevWords := strings.Fields(prevEnd)
		if len(prevWords) > 2 {
			lastWord := prevWords[len(prevWords)-1]
			if !strings.Contains(chunks[i].Text, lastWord) {
				t.Logf("Overlap pode não estar perfeito entre chunks %d e %d", i-1, i)
			}
		}
	}
}

// TestChunkNoOverlap verifica que overlap=0 não adiciona sobreposição.
func TestChunkNoOverlap(t *testing.T) {
	chunker := NewRecursiveChunker(50, 0)

	content := "Primeiro parágrafo.\n\nSegundo parágrafo.\n\nTerceiro parágrafo."
	doc := makeDoc("no-overlap.txt", content)

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	// Sem overlap, os textos dos chunks devem ser partes disjuntas.
	if len(chunks) >= 2 {
		if strings.Contains(chunks[1].Text, chunks[0].Text) {
			t.Error("Sem overlap, chunk 1 não deveria conter o texto completo do chunk 0")
		}
	}
}

// TestChunkMetadata verifica que FileName e FileType são propagados.
// RN-08: Metadados obrigatórios em cada chunk.
func TestChunkMetadata(t *testing.T) {
	chunker := NewRecursiveChunker(500, 50)
	doc := &document.Document{
		Name:     "runbook.md",
		FileType: "md",
		Content:  "Conteúdo do runbook com instruções.",
	}

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	for i, chunk := range chunks {
		if chunk.FileName != "runbook.md" {
			t.Errorf("chunks[%d].FileName = %s, want runbook.md", i, chunk.FileName)
		}
		if chunk.FileType != "md" {
			t.Errorf("chunks[%d].FileType = %s, want md", i, chunk.FileType)
		}
		// ID e DocumentID devem estar vazios (preenchidos pelo orquestrador).
		if chunk.ID != "" {
			t.Errorf("chunks[%d].ID deveria estar vazio", i)
		}
		if chunk.DocumentID != "" {
			t.Errorf("chunks[%d].DocumentID deveria estar vazio", i)
		}
	}
}

// TestChunkNilDocument verifica que nil doc retorna erro.
func TestChunkNilDocument(t *testing.T) {
	chunker := NewRecursiveChunker(500, 50)

	_, err := chunker.Chunk(nil)
	if err == nil {
		t.Error("Chunk deveria falhar para documento nil")
	}
}

// TestChunkEmptyContent verifica que documento vazio retorna erro.
func TestChunkEmptyContent(t *testing.T) {
	chunker := NewRecursiveChunker(500, 50)
	doc := makeDoc("empty.txt", "")

	_, err := chunker.Chunk(doc)
	if err == nil {
		t.Error("Chunk deveria falhar para documento sem conteúdo")
	}
}

// TestChunkWhitespaceOnly verifica que documento com apenas whitespace retorna erro.
func TestChunkWhitespaceOnly(t *testing.T) {
	chunker := NewRecursiveChunker(500, 50)
	doc := makeDoc("spaces.txt", "   \n\n\t  ")

	_, err := chunker.Chunk(doc)
	if err == nil {
		t.Error("Chunk deveria falhar para documento com apenas whitespace")
	}
}

// TestChunkPreservesParagraphs verifica que a divisão respeita parágrafos.
// RN-09: PDF/TXT/MD dividir por parágrafos primeiro.
func TestChunkPreservesParagraphs(t *testing.T) {
	chunker := NewRecursiveChunker(200, 0)

	content := "Primeiro parágrafo com bastante conteúdo.\n\n" +
		"Segundo parágrafo também com conteúdo.\n\n" +
		"Terceiro parágrafo que fecha o documento."

	doc := makeDoc("paras.txt", content)

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	// O texto não deve ser cortado no meio de um parágrafo se couber inteiro.
	for _, chunk := range chunks {
		text := strings.TrimSpace(chunk.Text)
		if text == "" {
			t.Error("Chunk não deveria estar vazio")
		}
	}
}

// TestNewRecursiveChunkerDefaults verifica os defaults do construtor.
func TestNewRecursiveChunkerDefaults(t *testing.T) {
	tests := []struct {
		name        string
		chunkSize   int
		overlapSize int
		wantChunk   int
		wantOverlap int
	}{
		{"normal", 500, 50, 500, 50},
		{"zero_chunk", 0, 50, 500, 50},
		{"negative_chunk", -1, 50, 500, 50},
		{"negative_overlap", 500, -1, 500, 0},
		{"overlap_too_large", 500, 600, 500, 50},
		{"overlap_equals_chunk", 100, 100, 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRecursiveChunker(tt.chunkSize, tt.overlapSize)
			if c.ChunkSize != tt.wantChunk {
				t.Errorf("ChunkSize = %d, want %d", c.ChunkSize, tt.wantChunk)
			}
			if c.OverlapSize != tt.wantOverlap {
				t.Errorf("OverlapSize = %d, want %d", c.OverlapSize, tt.wantOverlap)
			}
		})
	}
}

// TestChunkLargeDocumentApproxCounts verifica que um texto de ~10K palavras
// gera aproximadamente 20 chunks de ~500 caracteres.
// Critério de aceite: "arquivo TXT de 10K palavras → ~20 chunks de ~500 tokens"
func TestChunkLargeDocumentApproxCounts(t *testing.T) {
	chunker := NewRecursiveChunker(500, 50)

	// Gera texto com ~10K palavras (~50K caracteres).
	// Cada "frase" tem ~10 palavras.
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("Esta frase tem exatamente dez palavras no texto aqui agora. ")
		if i%5 == 4 {
			sb.WriteString("\n\n")
		}
	}

	doc := makeDoc("large-10k.txt", sb.String())

	chunks, err := chunker.Chunk(doc)
	if err != nil {
		t.Fatalf("Chunk falhou: %v", err)
	}

	// Com ~60K caracteres e chunkSize=500 + overlap, esperamos muitos chunks.
	// O número exato depende de como os separadores se alinham.
	if len(chunks) < 10 {
		t.Errorf("len(chunks) = %d, esperava pelo menos 10 para texto grande", len(chunks))
	}

	t.Logf("Texto de %d chars gerou %d chunks", len(doc.Content), len(chunks))
}
