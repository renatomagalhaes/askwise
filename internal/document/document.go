package document

import "io"

// Document representa o conteúdo extraído de um arquivo após parsing.
// Spec: design/02-MODELO-DADOS.md §4 (Document Conteúdo Parseado)
//
// O parser extrai texto do arquivo original e popula esta struct.
// O campo Sections é usado por parsers que identificam seções lógicas
// (ex: cabeçalhos em Markdown, chaves de primeiro nível em JSON/YAML).
type Document struct {
	// Name é o nome do arquivo fonte (ex: "manual.pdf").
	Name string

	// FileType é a extensão do arquivo sem ponto (ex: "pdf", "csv", "txt").
	FileType string

	// Content é o texto completo extraído do arquivo.
	// RN-03: Se vazio após extração, o documento deve ser rejeitado.
	Content string

	// Sections são divisões lógicas do conteúdo identificadas pelo parser.
	// Nem todos os formatos geram seções — depende da estrutura do arquivo.
	Sections []string
}

// Parser define a interface para extração de texto de arquivos.
// Spec: design/01-ARQUITETURA.md §2.3
//
// Cada formato de arquivo tem seu parser. A interface permite adicionar
// novos formatos sem alterar o código existente (Open/Closed Principle).
type Parser interface {
	// Parse lê o conteúdo do reader e extrai texto legível.
	// filename é usado para logging e metadados (não para abrir arquivo).
	// Retorna erro se o conteúdo não puder ser extraído.
	Parse(reader io.Reader, filename string) (*Document, error)

	// SupportedExtensions retorna as extensões de arquivo que este parser suporta.
	// Extensões sem ponto e em minúsculo (ex: ["txt", "md"]).
	SupportedExtensions() []string
}

// SupportedFormats lista todos os formatos de arquivo aceitos pelo sistema.
// RN-01: Apenas estes formatos são aceitos no upload.
var SupportedFormats = []string{"pdf", "csv", "txt", "yaml", "yml", "json", "md"}
