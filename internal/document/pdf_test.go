package document

import (
	"strings"
	"testing"
)

// TestPDFParserEmptyInput verifica que entrada vazia retorna Document sem conteúdo.
func TestPDFParserEmptyInput(t *testing.T) {
	parser := &PDFParser{}

	doc, err := parser.Parse(strings.NewReader(""), "empty.pdf")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.Content != "" {
		t.Errorf("Content = %q, want empty", doc.Content)
	}
}

// TestPDFParserInvalidContent verifica que conteúdo inválido retorna erro.
func TestPDFParserInvalidContent(t *testing.T) {
	parser := &PDFParser{}

	_, err := parser.Parse(strings.NewReader("isso não é um PDF"), "fake.pdf")
	if err == nil {
		t.Error("Parse deveria falhar para conteúdo não-PDF")
	}
}

// TestPDFParserSupportedExtensions verifica as extensões suportadas.
func TestPDFParserSupportedExtensions(t *testing.T) {
	parser := &PDFParser{}
	exts := parser.SupportedExtensions()

	if len(exts) != 1 || exts[0] != "pdf" {
		t.Errorf("SupportedExtensions = %v, want [pdf]", exts)
	}
}

// Nota: Testes com arquivos PDF reais seriam testes de integração.
// Para testes unitários, verificamos o comportamento com entradas edge-case.
// Um teste com PDF real pode ser adicionado em testdata/ futuramente.
