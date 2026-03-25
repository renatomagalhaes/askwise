package rag

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/renatomagalhaes/askwise/internal/chunker"
	"github.com/renatomagalhaes/askwise/internal/document"
	"github.com/renatomagalhaes/askwise/internal/llm"
	"github.com/renatomagalhaes/askwise/internal/retriever"
	"github.com/renatomagalhaes/askwise/internal/storage"
	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

// =============================================================================
// Mocks — implementações simples das interfaces para testes unitários.
// Cada mock registra as chamadas recebidas para permitir assertivas.
// =============================================================================

type mockParser struct {
	parseFn func(reader io.Reader, filename string) (*document.Document, error)
}

func (m *mockParser) Parse(reader io.Reader, filename string) (*document.Document, error) {
	return m.parseFn(reader, filename)
}
func (m *mockParser) SupportedExtensions() []string { return []string{"txt", "md"} }

type mockChunker struct {
	chunkFn func(doc *document.Document) ([]chunker.Chunk, error)
}

func (m *mockChunker) Chunk(doc *document.Document) ([]chunker.Chunk, error) {
	return m.chunkFn(doc)
}

type mockEmbedder struct {
	embedFn func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = make([]float32, 4)
	}
	return vecs, nil
}

func (m *mockEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return make([]float32, 4), nil
}

type mockVectorStore struct {
	upsertedPoints []vectorstore.Point
	deletedDocIDs  []string
}

func (m *mockVectorStore) EnsureCollection(_ context.Context, _ string, _ int) error { return nil }
func (m *mockVectorStore) Upsert(_ context.Context, _ string, points []vectorstore.Point) error {
	m.upsertedPoints = append(m.upsertedPoints, points...)
	return nil
}
func (m *mockVectorStore) Search(_ context.Context, _ string, _ []float32, _ int) ([]vectorstore.SearchResult, error) {
	return nil, nil
}
func (m *mockVectorStore) DeleteByDocID(_ context.Context, _ string, docID string) error {
	m.deletedDocIDs = append(m.deletedDocIDs, docID)
	return nil
}

type mockRetriever struct {
	retrieveFn func(ctx context.Context, query string, topK int) ([]retriever.RetrievedChunk, error)
}

func (m *mockRetriever) Retrieve(ctx context.Context, query string, topK int) ([]retriever.RetrievedChunk, error) {
	if m.retrieveFn != nil {
		return m.retrieveFn(ctx, query, topK)
	}
	return nil, nil
}

type mockLLM struct {
	chatFn           func(ctx context.Context, messages []llm.Message) (string, error)
	receivedMessages []llm.Message
}

func (m *mockLLM) ChatCompletion(ctx context.Context, messages []llm.Message) (string, error) {
	m.receivedMessages = messages
	if m.chatFn != nil {
		return m.chatFn(ctx, messages)
	}
	return "resposta mock", nil
}

type mockStorage struct {
	docs map[string]*storage.DocumentMeta
}

func newMockStorage() *mockStorage {
	return &mockStorage{docs: make(map[string]*storage.DocumentMeta)}
}

func (m *mockStorage) SaveDocument(_ context.Context, doc *storage.DocumentMeta) error {
	m.docs[doc.ID] = doc
	return nil
}
func (m *mockStorage) GetDocument(_ context.Context, id string) (*storage.DocumentMeta, error) {
	if doc, ok := m.docs[id]; ok {
		return doc, nil
	}
	return nil, nil
}
func (m *mockStorage) ListDocuments(_ context.Context) ([]storage.DocumentMeta, error) {
	var result []storage.DocumentMeta
	for _, d := range m.docs {
		result = append(result, *d)
	}
	return result, nil
}
func (m *mockStorage) DeleteDocument(_ context.Context, id string) error {
	delete(m.docs, id)
	return nil
}
func (m *mockStorage) GetDocumentByName(_ context.Context, name string) (*storage.DocumentMeta, error) {
	for _, d := range m.docs {
		if d.Name == name {
			return d, nil
		}
	}
	return nil, nil
}
func (m *mockStorage) Close() error { return nil }

// =============================================================================
// Helper para criar uma instância de RAG com todos os mocks
// =============================================================================

type testRAG struct {
	rag     *RAG
	parser  *mockParser
	chunker *mockChunker
	embed   *mockEmbedder
	vs      *mockVectorStore
	ret     *mockRetriever
	llm     *mockLLM
	storage *mockStorage
}

func newTestRAG() *testRAG {
	p := &mockParser{
		parseFn: func(reader io.Reader, filename string) (*document.Document, error) {
			data, _ := io.ReadAll(reader)
			return &document.Document{
				Name:     filename,
				FileType: "txt",
				Content:  string(data),
			}, nil
		},
	}

	c := &mockChunker{
		chunkFn: func(doc *document.Document) ([]chunker.Chunk, error) {
			return []chunker.Chunk{
				{Index: 0, Text: doc.Content, FileName: doc.Name, FileType: doc.FileType},
			}, nil
		},
	}

	e := &mockEmbedder{}
	vs := &mockVectorStore{}
	ret := &mockRetriever{}
	l := &mockLLM{}
	s := newMockStorage()

	r := NewRAG(p, c, e, vs, ret, l, s, Config{Collection: "test"})

	return &testRAG{
		rag: r, parser: p, chunker: c, embed: e,
		vs: vs, ret: ret, llm: l, storage: s,
	}
}

// =============================================================================
// Testes do Ingest
// =============================================================================

// TestIngestBasic verifica o fluxo completo de ingestão.
func TestIngestBasic(t *testing.T) {
	tr := newTestRAG()
	ctx := context.Background()

	result, err := tr.rag.Ingest(ctx, strings.NewReader("Conteúdo do documento"), "manual.txt", 1024)
	if err != nil {
		t.Fatalf("Ingest falhou: %v", err)
	}

	if result.DocumentID == "" {
		t.Error("DocumentID deveria ter sido gerado")
	}
	if result.ChunkCount != 1 {
		t.Errorf("ChunkCount = %d, want 1", result.ChunkCount)
	}

	// Verifica que o VectorStore recebeu os pontos.
	if len(tr.vs.upsertedPoints) != 1 {
		t.Fatalf("VectorStore recebeu %d pontos, want 1", len(tr.vs.upsertedPoints))
	}

	point := tr.vs.upsertedPoints[0]
	if point.Payload["document_id"] != result.DocumentID {
		t.Error("payload.document_id não corresponde ao DocumentID")
	}

	// Verifica que o Storage recebeu os metadados com status "ready".
	doc := tr.storage.docs[result.DocumentID]
	if doc == nil {
		t.Fatal("documento não foi salvo no Storage")
	}
	if doc.Status != "ready" {
		t.Errorf("Status = %s, want ready", doc.Status)
	}
}

// TestIngestMultipleChunks verifica ingestão com múltiplos chunks.
func TestIngestMultipleChunks(t *testing.T) {
	tr := newTestRAG()
	tr.chunker.chunkFn = func(doc *document.Document) ([]chunker.Chunk, error) {
		return []chunker.Chunk{
			{Index: 0, Text: "chunk 0", FileName: doc.Name, FileType: doc.FileType},
			{Index: 1, Text: "chunk 1", FileName: doc.Name, FileType: doc.FileType},
			{Index: 2, Text: "chunk 2", FileName: doc.Name, FileType: doc.FileType},
		}, nil
	}

	result, err := tr.rag.Ingest(context.Background(), strings.NewReader("texto longo"), "big.txt", 5000)
	if err != nil {
		t.Fatalf("Ingest falhou: %v", err)
	}

	if result.ChunkCount != 3 {
		t.Errorf("ChunkCount = %d, want 3", result.ChunkCount)
	}

	if len(tr.vs.upsertedPoints) != 3 {
		t.Errorf("VectorStore recebeu %d pontos, want 3", len(tr.vs.upsertedPoints))
	}
}

// TestIngestDuplicateReplacement verifica que re-upload substitui o anterior.
// RN-04: Duplicatas são substituídas. RN-22: Idempotência.
func TestIngestDuplicateReplacement(t *testing.T) {
	tr := newTestRAG()
	ctx := context.Background()

	// Primeiro upload.
	result1, err := tr.rag.Ingest(ctx, strings.NewReader("versão 1"), "doc.txt", 100)
	if err != nil {
		t.Fatalf("Primeiro Ingest falhou: %v", err)
	}

	// Segundo upload do mesmo arquivo.
	result2, err := tr.rag.Ingest(ctx, strings.NewReader("versão 2"), "doc.txt", 200)
	if err != nil {
		t.Fatalf("Segundo Ingest falhou: %v", err)
	}

	// IDs devem ser diferentes (novo UUID a cada upload).
	if result1.DocumentID == result2.DocumentID {
		t.Error("Re-upload deveria gerar novo DocumentID")
	}

	// O antigo deve ter sido deletado do VectorStore.
	if len(tr.vs.deletedDocIDs) != 1 {
		t.Fatalf("VectorStore deveria ter 1 delete, got %d", len(tr.vs.deletedDocIDs))
	}
	if tr.vs.deletedDocIDs[0] != result1.DocumentID {
		t.Error("VectorStore deletou o docID errado")
	}
}

// TestIngestSanitizesFilename verifica a sanitização do nome do arquivo.
// RN-05: Nome sanitizado.
func TestIngestSanitizesFilename(t *testing.T) {
	tr := newTestRAG()

	result, err := tr.rag.Ingest(context.Background(), strings.NewReader("conteúdo"), "Meu Documento (v2).txt", 100)
	if err != nil {
		t.Fatalf("Ingest falhou: %v", err)
	}

	doc := tr.storage.docs[result.DocumentID]
	if doc == nil {
		t.Fatal("documento não foi salvo")
	}

	// Nome deve ser sanitizado: sem espaços, caracteres especiais, lowercase.
	if strings.Contains(doc.Name, " ") {
		t.Errorf("Nome não foi sanitizado (tem espaço): %s", doc.Name)
	}
	if doc.Name != strings.ToLower(doc.Name) {
		t.Errorf("Nome não está em lowercase: %s", doc.Name)
	}
}

// TestIngestChunkIDs verifica que os IDs dos chunks são determinísticos.
// design/02-MODELO-DADOS.md §6: UUID v5 baseado em doc_id + chunk_index.
func TestIngestChunkIDs(t *testing.T) {
	docID := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"

	id0 := ChunkID(docID, 0)
	id1 := ChunkID(docID, 1)

	// IDs devem ser diferentes para índices diferentes.
	if id0 == id1 {
		t.Error("ChunkIDs para índices diferentes devem ser diferentes")
	}

	// IDs devem ser determinísticos (mesmo input → mesmo output).
	id0Again := ChunkID(docID, 0)
	if id0 != id0Again {
		t.Error("ChunkID deveria ser determinístico")
	}
}

// =============================================================================
// Testes do Query
// =============================================================================

// TestQueryBasic verifica o fluxo básico de consulta.
func TestQueryBasic(t *testing.T) {
	tr := newTestRAG()
	tr.ret.retrieveFn = func(_ context.Context, _ string, _ int) ([]retriever.RetrievedChunk, error) {
		return []retriever.RetrievedChunk{
			{Text: "Texto relevante", FileName: "manual.md", FileType: "md", Score: 0.9, Index: 0},
		}, nil
	}
	tr.llm.chatFn = func(_ context.Context, _ []llm.Message) (string, error) {
		return "A resposta baseada no contexto.", nil
	}

	resp, err := tr.rag.Query(context.Background(), "Como resolver o erro?", nil)
	if err != nil {
		t.Fatalf("Query falhou: %v", err)
	}

	if resp.Answer != "A resposta baseada no contexto." {
		t.Errorf("Answer = %q, want resposta do mock", resp.Answer)
	}

	if len(resp.Sources) != 1 {
		t.Fatalf("len(Sources) = %d, want 1", len(resp.Sources))
	}

	if resp.Sources[0].FileName != "manual.md" {
		t.Errorf("Sources[0].FileName = %s, want manual.md", resp.Sources[0].FileName)
	}
}

// TestQueryNoChunks verifica resposta quando não há chunks relevantes.
// RN-17: Informar ao usuário que não encontrou informações.
func TestQueryNoChunks(t *testing.T) {
	tr := newTestRAG()
	tr.ret.retrieveFn = func(_ context.Context, _ string, _ int) ([]retriever.RetrievedChunk, error) {
		return []retriever.RetrievedChunk{}, nil
	}

	resp, err := tr.rag.Query(context.Background(), "algo completamente fora da base", nil)
	if err != nil {
		t.Fatalf("Query falhou: %v", err)
	}

	if !strings.Contains(resp.Answer, "Não encontrei informações") {
		t.Errorf("Answer deveria conter mensagem de 'não encontrei', got: %s", resp.Answer)
	}

	if resp.Sources != nil {
		t.Error("Sources deveria ser nil quando não há chunks")
	}
}

// TestQuerySystemPromptIncludesContext verifica que o system prompt contém o contexto.
// RN-13: Resposta baseada em contexto.
func TestQuerySystemPromptIncludesContext(t *testing.T) {
	tr := newTestRAG()
	tr.ret.retrieveFn = func(_ context.Context, _ string, _ int) ([]retriever.RetrievedChunk, error) {
		return []retriever.RetrievedChunk{
			{Text: "O erro 5032 ocorre quando o timeout expira", FileName: "runbook.md", Score: 0.9},
		}, nil
	}

	tr.rag.Query(context.Background(), "O que é o erro 5032?", nil)

	// Verifica que o system prompt contém o contexto.
	if len(tr.llm.receivedMessages) == 0 {
		t.Fatal("LLM não recebeu mensagens")
	}

	sysPrompt := tr.llm.receivedMessages[0].Content
	if !strings.Contains(sysPrompt, "erro 5032 ocorre quando") {
		t.Error("System prompt deveria conter o texto do chunk")
	}
	if !strings.Contains(sysPrompt, "runbook.md") {
		t.Error("System prompt deveria citar o nome do arquivo fonte")
	}
}

// TestQueryWithHistory verifica que o histórico é incluído nas mensagens.
// RN-18: Manter últimas N mensagens como contexto.
func TestQueryWithHistory(t *testing.T) {
	tr := newTestRAG()
	tr.ret.retrieveFn = func(_ context.Context, _ string, _ int) ([]retriever.RetrievedChunk, error) {
		return []retriever.RetrievedChunk{
			{Text: "contexto", FileName: "doc.md", Score: 0.8},
		}, nil
	}

	history := []llm.Message{
		{Role: llm.RoleUser, Content: "primeira pergunta"},
		{Role: llm.RoleAssistant, Content: "primeira resposta"},
	}

	tr.rag.Query(context.Background(), "follow-up", history)

	// Mensagens: system + 2 histórico + 1 user = 4.
	if len(tr.llm.receivedMessages) != 4 {
		t.Fatalf("LLM recebeu %d mensagens, want 4", len(tr.llm.receivedMessages))
	}

	if tr.llm.receivedMessages[0].Role != llm.RoleSystem {
		t.Error("Primeira mensagem deveria ser system")
	}
	if tr.llm.receivedMessages[1].Content != "primeira pergunta" {
		t.Error("Segunda mensagem deveria ser o histórico")
	}
	if tr.llm.receivedMessages[3].Content != "follow-up" {
		t.Error("Última mensagem deveria ser a pergunta atual")
	}
}

// TestQueryDeduplicatesSources verifica que fontes duplicadas são mescladas.
func TestQueryDeduplicatesSources(t *testing.T) {
	tr := newTestRAG()
	tr.ret.retrieveFn = func(_ context.Context, _ string, _ int) ([]retriever.RetrievedChunk, error) {
		return []retriever.RetrievedChunk{
			{Text: "chunk 1", FileName: "manual.md", Score: 0.9, Index: 0},
			{Text: "chunk 2", FileName: "manual.md", Score: 0.8, Index: 3},
			{Text: "chunk 3", FileName: "faq.txt", Score: 0.7, Index: 1},
		}, nil
	}

	resp, err := tr.rag.Query(context.Background(), "query", nil)
	if err != nil {
		t.Fatalf("Query falhou: %v", err)
	}

	// 3 chunks de 2 arquivos → 2 fontes.
	if len(resp.Sources) != 2 {
		t.Errorf("len(Sources) = %d, want 2 (deduplicadas)", len(resp.Sources))
	}
}

// =============================================================================
// Testes do sanitizeFilename
// =============================================================================

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple.txt", "simple.txt"},
		{"My Document.pdf", "my-document.pdf"},
		{"file (copy).md", "file-copy.md"},
		{"UPPER.TXT", "upper.txt"},
		{"path/to/file.csv", "file.csv"},
		{"special@#$chars!.json", "specialchars.json"},
		{"multiple---dashes.yml", "multiple-dashes.yml"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
