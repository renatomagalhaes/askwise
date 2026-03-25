package document

import (
	"strings"
	"testing"
)

func TestStructuredParserJSON(t *testing.T) {
	parser := &StructuredParser{}
	jsonContent := `{
	"titulo": "Manual de Instalação",
	"versao": "2.0",
	"passos": ["baixar", "instalar", "configurar"]
}`

	doc, err := parser.Parse(strings.NewReader(jsonContent), "config.json")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"Name", doc.Name, "config.json"},
		{"FileType", doc.FileType, "json"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}

	// JSON com 3 chaves de primeiro nível → 3 seções.
	if len(doc.Sections) != 3 {
		t.Errorf("Sections = %d, want 3", len(doc.Sections))
	}

	if !strings.Contains(doc.Content, "titulo: Manual de Instalação") {
		t.Error("Content deveria conter 'titulo: Manual de Instalação'")
	}
	if !strings.Contains(doc.Content, "versao: 2.0") {
		t.Error("Content deveria conter 'versao: 2.0'")
	}
}

func TestStructuredParserJSONArray(t *testing.T) {
	parser := &StructuredParser{}
	jsonContent := `[
	{"nome": "Alice", "idade": 30},
	{"nome": "Bob", "idade": 25}
]`

	doc, err := parser.Parse(strings.NewReader(jsonContent), "users.json")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if len(doc.Sections) != 2 {
		t.Errorf("Sections = %d, want 2 (um por item do array)", len(doc.Sections))
	}

	if !strings.Contains(doc.Content, "Alice") {
		t.Error("Content deveria conter 'Alice'")
	}
}

func TestStructuredParserJSONSimpleValue(t *testing.T) {
	parser := &StructuredParser{}

	doc, err := parser.Parse(strings.NewReader(`"apenas uma string"`), "simple.json")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.Content != `"apenas uma string"` {
		t.Errorf("Content = %q, esperava valor simples preservado", doc.Content)
	}
}

func TestStructuredParserJSONInvalid(t *testing.T) {
	parser := &StructuredParser{}

	// JSON inválido — o parser deve lidar gracefully.
	doc, err := parser.Parse(strings.NewReader("{invalid json}"), "bad.json")
	if err != nil {
		t.Fatalf("Parse falhou para JSON inválido: %v", err)
	}

	// JSON inválido é tratado como texto simples.
	if doc.Content == "" {
		t.Error("Content deveria ter o conteúdo original para JSON inválido")
	}
}

func TestStructuredParserYAML(t *testing.T) {
	parser := &StructuredParser{}
	yamlContent := `titulo: Manual de Deploy
versao: "3.0"
passos:
  - baixar
  - instalar
  - configurar
notas:
  ambiente: produção
  timeout: 30s`

	doc, err := parser.Parse(strings.NewReader(yamlContent), "deploy.yaml")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"Name", doc.Name, "deploy.yaml"},
		{"FileType", doc.FileType, "yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}

	// 4 chaves de primeiro nível: titulo, versao, passos, notas.
	if len(doc.Sections) != 4 {
		t.Errorf("Sections = %d, want 4", len(doc.Sections))
	}

	if !strings.Contains(doc.Content, "titulo: Manual de Deploy") {
		t.Error("Content deveria conter o conteúdo YAML original")
	}
}

func TestStructuredParserYML(t *testing.T) {
	parser := &StructuredParser{}
	yamlContent := `key: value`

	doc, err := parser.Parse(strings.NewReader(yamlContent), "config.yml")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.FileType != "yml" {
		t.Errorf("FileType = %s, want yml", doc.FileType)
	}
}

func TestStructuredParserSupportedExtensions(t *testing.T) {
	parser := &StructuredParser{}
	exts := parser.SupportedExtensions()

	expected := map[string]bool{"json": true, "yaml": true, "yml": true}
	for _, ext := range exts {
		if !expected[ext] {
			t.Errorf("extensão inesperada: %s", ext)
		}
		delete(expected, ext)
	}
	if len(expected) > 0 {
		t.Errorf("extensões faltando: %v", expected)
	}
}
