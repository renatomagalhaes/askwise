// Package vectorstore gerencia o armazenamento e busca de embeddings vetoriais
// no Qdrant via REST API.
//
// ADR-004: Qdrant foi escolhido por ter REST API simples, Docker fácil e ser grátis.
// Spec dirigindo: RF-05 (armazenamento), RN-08 (metadados), RN-21 (integridade)
//
// Este pacote usa a REST API do Qdrant diretamente via net/http, evitando
// a dependência pesada do client gRPC (protobuf, grpc). Isso mantém o
// projeto alinhado com RNF-03 (simplicidade) e facilita o entendimento
// educativo das operações vetoriais.
//
// Operações suportadas:
//   - CreateCollection: cria a collection com dimensão e métrica configuradas
//   - Upsert: insere ou atualiza pontos (vetores + payload)
//   - Search: busca os K pontos mais similares a um vetor de consulta
//   - DeleteByDocID: remove todos os pontos de um documento específico
package vectorstore
