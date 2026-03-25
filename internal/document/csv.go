package document

import (
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// CSVParser converte arquivos CSV em texto estruturado.
// RF-02: CSV converte linhas em texto estruturado (cabeçalho + valores).
// RN-09: Cada grupo de linhas pode virar um chunk (com cabeçalho repetido).
type CSVParser struct{}

// Parse lê o conteúdo CSV e converte para texto legível.
// Cada linha é convertida em formato "campo: valor" usando o cabeçalho.
// Se não houver cabeçalho (arquivo com 0-1 linhas), usa o conteúdo direto.
func (p *CSVParser) Parse(reader io.Reader, filename string) (*Document, error) {
	csvReader := csv.NewReader(reader)
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear CSV %s: %w", filename, err)
	}

	if len(records) == 0 {
		return &Document{
			Name:     filename,
			FileType: "csv",
			Content:  "",
		}, nil
	}

	headers := records[0]
	var content strings.Builder
	var sections []string

	// Cada linha de dados vira uma seção com formato "campo: valor".
	// O cabeçalho é repetido em cada seção para facilitar o chunking (RN-09).
	for i := 1; i < len(records); i++ {
		row := records[i]
		var section strings.Builder
		section.WriteString(fmt.Sprintf("Registro %d:\n", i))

		for j, header := range headers {
			value := ""
			if j < len(row) {
				value = strings.TrimSpace(row[j])
			}
			section.WriteString(fmt.Sprintf("  %s: %s\n", header, value))
		}

		sectionStr := section.String()
		sections = append(sections, strings.TrimSpace(sectionStr))

		content.WriteString(sectionStr)
		content.WriteString("\n")
	}

	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	ext = strings.ToLower(ext)

	return &Document{
		Name:     filename,
		FileType: ext,
		Content:  strings.TrimSpace(content.String()),
		Sections: sections,
	}, nil
}

// SupportedExtensions retorna as extensões suportadas pelo CSVParser.
func (p *CSVParser) SupportedExtensions() []string {
	return []string{"csv"}
}
