package document

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	// ledongthuc/pdf é uma biblioteca pure Go para extração de texto de PDFs.
	// Listada em AGENTS.md como dependência esperada do projeto.
	"github.com/ledongthuc/pdf"
)

// PDFParser extrai texto de arquivos PDF.
// RF-02: Extrair texto de todas as páginas (sem OCR nesta PoC).
//
// Limitação conhecida: PDFs baseados em imagem (scanned) não terão texto extraído.
// Para estes casos, o conteúdo será vazio e o upload será rejeitado (RN-03).
type PDFParser struct{}

// Parse lê o conteúdo PDF e extrai texto de todas as páginas.
// O reader é buffered em memória pois a lib PDF precisa de io.ReaderAt.
func (p *PDFParser) Parse(reader io.Reader, filename string) (*Document, error) {
	// Precisa ler tudo em memória pois pdf.NewReader exige io.ReaderAt + tamanho.
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo PDF %s: %w", filename, err)
	}

	if len(data) == 0 {
		return &Document{
			Name:     filename,
			FileType: "pdf",
			Content:  "",
		}, nil
	}

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir PDF %s: %w", filename, err)
	}

	var content strings.Builder
	var sections []string
	numPages := r.NumPage()

	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		text, err := extractPageText(page)
		if err != nil {
			// Loga mas não falha — páginas individuais podem ter problemas.
			continue
		}

		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}

		sections = append(sections, text)
		content.WriteString(text)
		content.WriteString("\n\n")
	}

	return &Document{
		Name:     filename,
		FileType: "pdf",
		Content:  strings.TrimSpace(content.String()),
		Sections: sections,
	}, nil
}

// SupportedExtensions retorna as extensões suportadas pelo PDFParser.
func (p *PDFParser) SupportedExtensions() []string {
	return []string{"pdf"}
}

// extractPageText extrai o texto de uma página PDF.
// Itera pelas rows e words da página para reconstruir o texto.
func extractPageText(page pdf.Page) (string, error) {
	rows, err := page.GetTextByRow()
	if err != nil {
		return "", fmt.Errorf("falha ao extrair texto da página: %w", err)
	}

	var sb strings.Builder
	for _, row := range rows {
		for i, word := range row.Content {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(word.S)
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
