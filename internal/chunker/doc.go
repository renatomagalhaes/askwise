// Package chunker divide documentos em pedaços menores (chunks) para indexação vetorial.
//
// RF-03: O sistema deve dividir o texto extraído em chunks de ~500 tokens com overlap de ~50 tokens.
// Spec dirigindo: RN-06 (tamanho), RN-07 (overlap), RN-08 (metadados), RN-09 (formato).
//
// A estratégia de splitting é "recursive character text splitting":
// tenta dividir pelo separador mais significativo primeiro (parágrafos),
// e recursivamente usa separadores menores (linhas, frases, palavras) quando necessário.
//
// Os valores de chunk_size e overlap são configuráveis via variáveis de ambiente
// (RAG_CHUNK_SIZE e RAG_CHUNK_OVERLAP). Nesta PoC, usamos contagem de caracteres
// como aproximação prática de tokens (1 token ≈ 4 caracteres em inglês/português).
package chunker
