package document

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// TextParser extrai texto de arquivos de texto puro: TXT e Markdown.
// RF-02: TXT usa conteúdo diretamente; MD usa conteúdo diretamente
// (a remoção de sintaxe Markdown é opcional conforme spec).
type TextParser struct{}

// Parse lê o conteúdo do reader e retorna um Document com o texto extraído.
// Para TXT e MD, o conteúdo é usado diretamente sem transformação.
func (p *TextParser) Parse(reader io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo %s: %w", filename, err)
	}

	content := strings.TrimSpace(string(data))

	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	ext = strings.ToLower(ext)

	doc := &Document{
		Name:     filename,
		FileType: ext,
		Content:  content,
	}

	// Para Markdown, identifica seções pelos cabeçalhos (# ou ##).
	// RN-09: Estratégia de chunking pode usar seções para preservar estrutura.
	if ext == "md" {
		doc.Sections = extractMarkdownSections(content)
	}

	return doc, nil
}

// SupportedExtensions retorna as extensões suportadas pelo TextParser.
func (p *TextParser) SupportedExtensions() []string {
	return []string{"txt", "md"}
}

// extractMarkdownSections divide o conteúdo Markdown pelas linhas de cabeçalho.
// Cada seção começa com um cabeçalho (#, ##, ###, etc.) e vai até o próximo.
func extractMarkdownSections(content string) []string {
	lines := strings.Split(content, "\n")
	var sections []string
	var current strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") && current.Len() > 0 {
			sections = append(sections, strings.TrimSpace(current.String()))
			current.Reset()
		}
		current.WriteString(line)
		current.WriteString("\n")
	}

	if current.Len() > 0 {
		section := strings.TrimSpace(current.String())
		if section != "" {
			sections = append(sections, section)
		}
	}

	return sections
}
