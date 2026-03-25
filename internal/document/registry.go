package document

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
)

// Registry centraliza o roteamento de arquivos para o parser correto.
// RN-01: Apenas formatos registrados são aceitos.
//
// O Registry implementa a interface Parser, funcionando como um facade
// que detecta o formato pelo nome do arquivo e delega ao parser específico.
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry cria um Registry com todos os parsers padrão registrados.
// Cada parser registra as extensões que suporta.
func NewRegistry() *Registry {
	r := &Registry{
		parsers: make(map[string]Parser),
	}

	r.Register(&TextParser{})
	r.Register(&CSVParser{})
	r.Register(&StructuredParser{})
	r.Register(&PDFParser{})

	slog.Info("document registry initialized",
		"component", "document",
		"formats", r.SupportedFormats(),
	)

	return r
}

// Register adiciona um parser ao registry para todas as suas extensões.
func (r *Registry) Register(p Parser) {
	for _, ext := range p.SupportedExtensions() {
		r.parsers[strings.ToLower(ext)] = p
	}
}

// Parse detecta o formato do arquivo e delega ao parser correto.
// Implementa a interface Parser para que o Registry possa ser usado
// como um parser universal pelo restante do sistema.
func (r *Registry) Parse(reader io.Reader, filename string) (*Document, error) {
	parser, err := r.GetParser(filename)
	if err != nil {
		return nil, err
	}

	slog.Info("parsing document",
		"component", "document",
		"filename", filename,
		"parser", fmt.Sprintf("%T", parser),
	)

	doc, err := parser.Parse(reader, filename)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear %s: %w", filename, err)
	}

	// RN-03: Rejeitar arquivo sem conteúdo textual extraível.
	if strings.TrimSpace(doc.Content) == "" {
		return nil, fmt.Errorf("arquivo %s não contém conteúdo textual extraível", filename)
	}

	slog.Info("document parsed",
		"component", "document",
		"filename", filename,
		"content_length", len(doc.Content),
		"sections", len(doc.Sections),
	)

	return doc, nil
}

// SupportedExtensions retorna todas as extensões registradas.
func (r *Registry) SupportedExtensions() []string {
	return r.SupportedFormats()
}

// GetParser retorna o parser apropriado para o arquivo.
// RN-01: Retorna erro se o formato não for suportado.
func (r *Registry) GetParser(filename string) (Parser, error) {
	ext := extractExtension(filename)
	parser, ok := r.parsers[ext]
	if !ok {
		return nil, fmt.Errorf(
			"formato não suportado: .%s (formatos aceitos: %s)",
			ext,
			strings.Join(r.SupportedFormats(), ", "),
		)
	}
	return parser, nil
}

// SupportedFormats retorna a lista de extensões suportadas, em ordem alfabética.
func (r *Registry) SupportedFormats() []string {
	seen := make(map[string]bool)
	var formats []string
	for ext := range r.parsers {
		if !seen[ext] {
			seen[ext] = true
			formats = append(formats, ext)
		}
	}
	sort.Strings(formats)
	return formats
}

// IsSupported verifica se uma extensão de arquivo é suportada.
func (r *Registry) IsSupported(filename string) bool {
	ext := extractExtension(filename)
	_, ok := r.parsers[ext]
	return ok
}

// extractExtension extrai a extensão do arquivo, sem ponto, em minúsculo.
func extractExtension(filename string) string {
	ext := filepath.Ext(filename)
	ext = strings.TrimPrefix(ext, ".")
	return strings.ToLower(ext)
}
