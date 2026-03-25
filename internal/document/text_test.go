package document

import (
	"strings"
	"testing"
)

func TestTextParserTXT(t *testing.T) {
	parser := &TextParser{}
	content := "Linha 1 do documento.\nLinha 2 do documento.\nLinha 3 com mais detalhes."

	doc, err := parser.Parse(strings.NewReader(content), "readme.txt")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"Name", doc.Name, "readme.txt"},
		{"FileType", doc.FileType, "txt"},
		{"Content", doc.Content, content},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}

	// TXT não deve gerar seções.
	if len(doc.Sections) != 0 {
		t.Errorf("Sections = %d, want 0 para TXT", len(doc.Sections))
	}
}

func TestTextParserMarkdown(t *testing.T) {
	parser := &TextParser{}
	content := `# Título Principal

Este é o conteúdo da primeira seção.

## Segunda Seção

Conteúdo da segunda seção com mais detalhes.

## Terceira Seção

Conteúdo final.`

	doc, err := parser.Parse(strings.NewReader(content), "manual.md")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.FileType != "md" {
		t.Errorf("FileType = %s, want md", doc.FileType)
	}

	if doc.Content != content {
		t.Errorf("Content não preservou o conteúdo original")
	}

	// Markdown deve ter seções divididas pelos cabeçalhos.
	if len(doc.Sections) != 3 {
		t.Errorf("Sections = %d, want 3", len(doc.Sections))
	}
}

func TestTextParserMarkdownSingleSection(t *testing.T) {
	parser := &TextParser{}
	content := "Texto sem cabeçalhos markdown.\nApenas parágrafos simples."

	doc, err := parser.Parse(strings.NewReader(content), "notes.md")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	// Sem cabeçalhos, todo o conteúdo é uma única seção.
	if len(doc.Sections) != 1 {
		t.Errorf("Sections = %d, want 1", len(doc.Sections))
	}
}

func TestTextParserTrimsWhitespace(t *testing.T) {
	parser := &TextParser{}
	content := "   \n\n  Conteúdo com espaços ao redor.  \n\n   "

	doc, err := parser.Parse(strings.NewReader(content), "spaces.txt")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.Content != "Conteúdo com espaços ao redor." {
		t.Errorf("Content = %q, esperava trim de whitespace", doc.Content)
	}
}

func TestTextParserEmptyContent(t *testing.T) {
	parser := &TextParser{}

	doc, err := parser.Parse(strings.NewReader(""), "empty.txt")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.Content != "" {
		t.Errorf("Content = %q, want empty", doc.Content)
	}
}

func TestTextParserSupportedExtensions(t *testing.T) {
	parser := &TextParser{}
	exts := parser.SupportedExtensions()

	expected := map[string]bool{"txt": true, "md": true}
	for _, ext := range exts {
		if !expected[ext] {
			t.Errorf("extensão inesperada: %s", ext)
		}
	}
	if len(exts) != 2 {
		t.Errorf("len(SupportedExtensions) = %d, want 2", len(exts))
	}
}
