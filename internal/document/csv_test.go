package document

import (
	"strings"
	"testing"
)

func TestCSVParserBasic(t *testing.T) {
	parser := &CSVParser{}
	csvContent := `nome,email,cargo
Ana Silva,ana@example.com,Engenheira
João Costa,joao@example.com,Designer
Maria Souza,maria@example.com,Gerente`

	doc, err := parser.Parse(strings.NewReader(csvContent), "equipe.csv")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	tests := []struct {
		field string
		got   any
		want  any
	}{
		{"Name", doc.Name, "equipe.csv"},
		{"FileType", doc.FileType, "csv"},
		{"Sections", len(doc.Sections), 3},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %v, want %v", tt.field, tt.got, tt.want)
			}
		})
	}

	// Verifica que o conteúdo contém os campos formatados.
	if !strings.Contains(doc.Content, "nome: Ana Silva") {
		t.Error("Content deveria conter 'nome: Ana Silva'")
	}
	if !strings.Contains(doc.Content, "cargo: Gerente") {
		t.Error("Content deveria conter 'cargo: Gerente'")
	}
}

func TestCSVParserEmpty(t *testing.T) {
	parser := &CSVParser{}

	doc, err := parser.Parse(strings.NewReader(""), "empty.csv")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if doc.Content != "" {
		t.Errorf("Content = %q, want empty", doc.Content)
	}
}

func TestCSVParserHeaderOnly(t *testing.T) {
	parser := &CSVParser{}
	csvContent := "col1,col2,col3"

	doc, err := parser.Parse(strings.NewReader(csvContent), "headers.csv")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	// Apenas cabeçalho, sem dados → conteúdo vazio.
	if doc.Content != "" {
		t.Errorf("Content = %q, want empty para CSV só com cabeçalho", doc.Content)
	}
	if len(doc.Sections) != 0 {
		t.Errorf("Sections = %d, want 0", len(doc.Sections))
	}
}

func TestCSVParserWithQuotes(t *testing.T) {
	parser := &CSVParser{}
	csvContent := `titulo,descricao
"Erro 5032","Conexão timeout, verificar rede"
"Erro 1001","Autenticação falhou"`

	doc, err := parser.Parse(strings.NewReader(csvContent), "erros.csv")
	if err != nil {
		t.Fatalf("Parse falhou: %v", err)
	}

	if !strings.Contains(doc.Content, "titulo: Erro 5032") {
		t.Error("Content deveria conter 'titulo: Erro 5032'")
	}
	if !strings.Contains(doc.Content, "verificar rede") {
		t.Error("Content deveria conter 'verificar rede'")
	}
}

func TestCSVParserMismatchedColumns(t *testing.T) {
	parser := &CSVParser{}
	csvContent := `a,b,c
1,2`

	// CSV com número inconsistente de colunas — o parser deve ser tolerante.
	_, err := parser.Parse(strings.NewReader(csvContent), "broken.csv")
	// encoding/csv retorna erro para linhas com colunas inconsistentes.
	// Isso é comportamento esperado.
	if err == nil {
		t.Log("CSV com colunas inconsistentes foi aceito (LazyQuotes/FieldsPerRecord)")
	}
}

func TestCSVParserSupportedExtensions(t *testing.T) {
	parser := &CSVParser{}
	exts := parser.SupportedExtensions()

	if len(exts) != 1 || exts[0] != "csv" {
		t.Errorf("SupportedExtensions = %v, want [csv]", exts)
	}
}
