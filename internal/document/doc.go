// Package document é responsável por extrair texto de diferentes formatos de arquivo.
//
// RF-02: O sistema deve extrair texto legível de cada formato suportado.
// Spec dirigindo: RF-01 (upload), RF-02 (extração), RN-01 (validação de formato),
// RN-03 (arquivo não vazio), RN-09 (chunking por formato).
//
// Cada formato tem seu próprio parser que implementa a interface Parser.
// O Registry centraliza a detecção de formato e roteamento para o parser correto.
//
// Parsers disponíveis:
//   - TextParser: TXT e MD (conteúdo direto)
//   - CSVParser: CSV (cabeçalho + valores em texto estruturado)
//   - StructuredParser: JSON e YAML (estrutura convertida em texto legível)
//   - PDFParser: PDF (extração de texto de todas as páginas, sem OCR)
package document
