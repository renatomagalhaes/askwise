package document

import (
	"strings"
	"testing"
)

func TestRegistryParseText(t *testing.T) {
	reg := NewRegistry()
	content := "Conteúdo de teste para o registry."

	doc, err := reg.Parse(strings.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.Content != content {
		t.Errorf("Content = %q, want %q", doc.Content, content)
	}
}

func TestRegistryParseMarkdown(t *testing.T) {
	reg := NewRegistry()
	content := "# Título\n\nConteúdo do documento."

	doc, err := reg.Parse(strings.NewReader(content), "doc.md")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.FileType != "md" {
		t.Errorf("FileType = %s, want md", doc.FileType)
	}
}

func TestRegistryParseCSV(t *testing.T) {
	reg := NewRegistry()
	content := "col1,col2\nval1,val2"

	doc, err := reg.Parse(strings.NewReader(content), "data.csv")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if !strings.Contains(doc.Content, "col1: val1") {
		t.Error("Content deveria conter 'col1: val1'")
	}
}

func TestRegistryParseJSON(t *testing.T) {
	reg := NewRegistry()
	content := `{"chave": "valor"}`

	doc, err := reg.Parse(strings.NewReader(content), "config.json")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if !strings.Contains(doc.Content, "chave: valor") {
		t.Error("Content deveria conter 'chave: valor'")
	}
}

func TestRegistryParseYAML(t *testing.T) {
	reg := NewRegistry()
	content := "chave: valor"

	doc, err := reg.Parse(strings.NewReader(content), "config.yaml")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.FileType != "yaml" {
		t.Errorf("FileType = %s, want yaml", doc.FileType)
	}
}

// TestRegistryUnsupportedFormat verifica que formatos não suportados geram erro.
// RN-01: Rejeitar formatos não listados.
func TestRegistryUnsupportedFormat(t *testing.T) {
	reg := NewRegistry()

	tests := []string{"image.png", "data.xlsx", "script.py", "archive.zip"}

	for _, filename := range tests {
		t.Run(filename, func(t *testing.T) {
			_, err := reg.Parse(strings.NewReader("content"), filename)
			if err == nil {
				t.Errorf("Parse deveria rejeitar formato não suportado: %s", filename)
			}
			if !strings.Contains(err.Error(), "formato não suportado") {
				t.Errorf("erro deveria mencionar 'formato não suportado', got: %s", err.Error())
			}
		})
	}
}

// TestRegistryEmptyContent verifica que RN-03 rejeita arquivos sem conteúdo.
func TestRegistryEmptyContent(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Parse(strings.NewReader(""), "empty.txt")
	if err == nil {
		t.Error("Parse deveria rejeitar arquivo sem conteúdo textual")
	}
	if !strings.Contains(err.Error(), "não contém conteúdo textual") {
		t.Errorf("erro deveria mencionar conteúdo vazio, got: %s", err.Error())
	}
}

// TestRegistryWhitespaceOnly verifica que whitespace-only é rejeitado.
func TestRegistryWhitespaceOnly(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Parse(strings.NewReader("   \n\n\t  "), "spaces.txt")
	if err == nil {
		t.Error("Parse deveria rejeitar arquivo com apenas whitespace")
	}
}

func TestRegistryIsSupported(t *testing.T) {
	reg := NewRegistry()

	tests := []struct {
		filename string
		want     bool
	}{
		{"doc.txt", true},
		{"doc.md", true},
		{"doc.csv", true},
		{"doc.json", true},
		{"doc.yaml", true},
		{"doc.yml", true},
		{"doc.pdf", true},
		{"doc.png", false},
		{"doc.xlsx", false},
		{"noextension", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := reg.IsSupported(tt.filename)
			if got != tt.want {
				t.Errorf("IsSupported(%s) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestRegistrySupportedFormats(t *testing.T) {
	reg := NewRegistry()
	formats := reg.SupportedFormats()

	// Deve conter todos os formatos listados em SupportedFormats.
	expected := map[string]bool{
		"csv": true, "json": true, "md": true,
		"pdf": true, "txt": true, "yaml": true, "yml": true,
	}

	for _, f := range formats {
		if !expected[f] {
			t.Errorf("formato inesperado: %s", f)
		}
		delete(expected, f)
	}

	if len(expected) > 0 {
		t.Errorf("formatos faltando: %v", expected)
	}
}

// TestRegistryCaseInsensitive verifica que extensões são case-insensitive.
func TestRegistryCaseInsensitive(t *testing.T) {
	reg := NewRegistry()

	tests := []string{"FILE.TXT", "Doc.MD", "data.CSV", "Config.JSON", "deploy.YAML"}

	for _, filename := range tests {
		t.Run(filename, func(t *testing.T) {
			if !reg.IsSupported(filename) {
				t.Errorf("IsSupported(%s) deveria ser true (case-insensitive)", filename)
			}
		})
	}
}
