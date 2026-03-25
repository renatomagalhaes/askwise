package document

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// StructuredParser converte arquivos JSON e YAML em texto legível.
// RF-02: JSON e YAML devem ser convertidos em texto legível.
// RN-09: Dividir por chaves de primeiro nível para preservar estrutura semântica.
//
// JSON é parseado com encoding/json (standard library).
// YAML é usado como texto direto — já é human-readable e evita dependência externa
// (RNF-03: mínimo de dependências).
type StructuredParser struct{}

// Parse lê o conteúdo JSON ou YAML e converte para texto legível.
func (p *StructuredParser) Parse(reader io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler arquivo %s: %w", filename, err)
	}

	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	ext = strings.ToLower(ext)

	var content string
	var sections []string

	switch ext {
	case "json":
		content, sections, err = parseJSON(data, filename)
		if err != nil {
			return nil, err
		}
	case "yaml", "yml":
		content, sections = parseYAML(data)
	default:
		return nil, fmt.Errorf("formato não suportado pelo StructuredParser: %s", ext)
	}

	return &Document{
		Name:     filename,
		FileType: ext,
		Content:  content,
		Sections: sections,
	}, nil
}

// SupportedExtensions retorna as extensões suportadas pelo StructuredParser.
func (p *StructuredParser) SupportedExtensions() []string {
	return []string{"json", "yaml", "yml"}
}

// parseJSON converte JSON em texto legível, extraindo chaves de primeiro nível
// como seções separadas.
func parseJSON(data []byte, filename string) (string, []string, error) {
	// Tenta parsear como objeto (caso mais comum para documentos).
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err == nil {
		return jsonObjectToText(obj)
	}

	// Tenta parsear como array.
	var arr []any
	if err := json.Unmarshal(data, &arr); err == nil {
		return jsonArrayToText(arr)
	}

	// Se não for objeto nem array, tenta como valor simples (string, number, etc.).
	content := strings.TrimSpace(string(data))
	return content, nil, nil
}

// jsonObjectToText converte um JSON object em texto, com cada chave de primeiro
// nível como uma seção separada (RN-09).
func jsonObjectToText(obj map[string]any) (string, []string, error) {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var content strings.Builder
	var sections []string

	for _, key := range keys {
		val := obj[key]
		formatted := formatJSONValue(val, 0)
		section := fmt.Sprintf("%s: %s", key, formatted)
		sections = append(sections, section)
		content.WriteString(section)
		content.WriteString("\n\n")
	}

	return strings.TrimSpace(content.String()), sections, nil
}

// jsonArrayToText converte um JSON array em texto, com cada elemento como seção.
func jsonArrayToText(arr []any) (string, []string, error) {
	var content strings.Builder
	var sections []string

	for i, item := range arr {
		formatted := formatJSONValue(item, 0)
		section := fmt.Sprintf("Item %d: %s", i+1, formatted)
		sections = append(sections, section)
		content.WriteString(section)
		content.WriteString("\n\n")
	}

	return strings.TrimSpace(content.String()), sections, nil
}

// formatJSONValue converte um valor JSON em texto legível com indentação.
func formatJSONValue(val any, depth int) string {
	indent := strings.Repeat("  ", depth)

	switch v := val.(type) {
	case map[string]any:
		if len(v) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var sb strings.Builder
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("\n%s  %s: %s", indent, k, formatJSONValue(v[k], depth+1)))
		}
		return sb.String()

	case []any:
		if len(v) == 0 {
			return "[]"
		}
		var sb strings.Builder
		for i, item := range v {
			sb.WriteString(fmt.Sprintf("\n%s  [%d] %s", indent, i+1, formatJSONValue(item, depth+1)))
		}
		return sb.String()

	case string:
		return v

	case nil:
		return "null"

	default:
		return fmt.Sprintf("%v", v)
	}
}

// parseYAML usa o conteúdo YAML diretamente — YAML já é human-readable.
// Seções são identificadas pelas chaves de primeiro nível (linhas sem indentação
// seguidas de ":").
func parseYAML(data []byte) (string, []string) {
	content := strings.TrimSpace(string(data))

	lines := strings.Split(content, "\n")
	var sections []string
	var current strings.Builder

	for _, line := range lines {
		// Chave de primeiro nível: linha sem indentação que contém ":"
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && line[0] != '-' && strings.Contains(line, ":") {
			if current.Len() > 0 {
				sections = append(sections, strings.TrimSpace(current.String()))
				current.Reset()
			}
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

	return content, sections
}
