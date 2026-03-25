package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/renatomagalhaes/askwise/internal/chunker"
	"github.com/renatomagalhaes/askwise/internal/config"
	"github.com/renatomagalhaes/askwise/internal/document"
	"github.com/renatomagalhaes/askwise/internal/llm"
	"github.com/renatomagalhaes/askwise/internal/rag"
	"github.com/renatomagalhaes/askwise/internal/retriever"
	"github.com/renatomagalhaes/askwise/internal/storage"
	"github.com/renatomagalhaes/askwise/internal/vectorstore"
)

// =============================================================================
// Mocks — as mesmas interfaces usadas em rag_test.go, recriadas aqui
// porque estamos em package main (cmd/server).
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

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = make([]float32, 4)
	}
	return vecs, nil
}
func (m *mockEmbedder) EmbedQuery(_ context.Context, _ string) ([]float32, error) {
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

type mockRetriever struct{}

func (m *mockRetriever) Retrieve(_ context.Context, _ string, _ int) ([]retriever.RetrievedChunk, error) {
	return nil, nil
}

type mockLLM struct{}

func (m *mockLLM) ChatCompletion(_ context.Context, _ []llm.Message) (string, error) {
	return "mock", nil
}

type mockStorage struct {
	docs map[string]*storage.DocumentMeta
}

func newMockStorage() *mockStorage {
	return &mockStorage{docs: make(map[string]*storage.DocumentMeta)}
}

func (m *mockStorage) SaveDocument(_ context.Context, doc *storage.DocumentMeta) error {
	now := time.Now().UTC()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now
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
// Helpers para montar o App de teste e executar requests
// =============================================================================

func newTestApp() (*App, *mockStorage, *mockVectorStore) {
	store := newMockStorage()
	vs := &mockVectorStore{}

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

	r := rag.NewRAG(p, c, &mockEmbedder{}, vs, &mockRetriever{}, &mockLLM{}, store,
		rag.Config{Collection: "test"},
	)

	cfg := &config.Config{
		ServerPort:   8484,
		MaxFileSize:  10 * 1024 * 1024,
		QdrantHost:   "localhost",
		QdrantPort:   6333,
		OpenAIAPIKey: "sk-test-key",
	}

	app := &App{
		rag:      r,
		storage:  store,
		registry: document.NewRegistry(),
		cfg:      cfg,
	}

	return app, store, vs
}

// createMultipartRequest monta um request multipart com o conteúdo dado.
func createMultipartRequest(t *testing.T, filename, content string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	part.Write([]byte(content))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// =============================================================================
// Testes do Health Check
// =============================================================================

func TestHealthEndpoint(t *testing.T) {
	app, _, _ := newTestApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", app.handleHealth())
	mux.ServeHTTP(rec, req)

	// Qdrant não está rodando no teste, então será unhealthy.
	// Mas o handler deve retornar JSON válido.
	var resp healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode health response: %v", err)
	}

	if resp.Version != version {
		t.Errorf("version = %s, want %s", resp.Version, version)
	}
	if resp.Dependencies["openai"] != "configured" {
		t.Errorf("openai = %s, want configured", resp.Dependencies["openai"])
	}
	if resp.Dependencies["sqlite"] != "connected" {
		t.Errorf("sqlite = %s, want connected", resp.Dependencies["sqlite"])
	}
}

// =============================================================================
// Testes de Upload (POST /api/v1/documents)
// =============================================================================

func TestUploadSuccess(t *testing.T) {
	app, store, _ := newTestApp()

	req := createMultipartRequest(t, "manual.txt", "Conteúdo do manual de teste")
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/documents", app.handleUpload())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp uploadResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.ID == "" {
		t.Error("ID deveria ter sido gerado")
	}
	if resp.Name != "manual.txt" {
		t.Errorf("Name = %s, want manual.txt", resp.Name)
	}
	if resp.ChunkCount != 1 {
		t.Errorf("ChunkCount = %d, want 1", resp.ChunkCount)
	}
	if resp.Status != "ready" {
		t.Errorf("Status = %s, want ready", resp.Status)
	}
	if !strings.Contains(resp.Message, "1 chunks indexados") {
		t.Errorf("Message = %s, want mention of chunks", resp.Message)
	}

	// Verifica que o documento foi salvo no storage.
	if len(store.docs) != 1 {
		t.Errorf("storage docs = %d, want 1", len(store.docs))
	}
}

func TestUploadUnsupportedFormat(t *testing.T) {
	app, _, _ := newTestApp()

	req := createMultipartRequest(t, "planilha.docx", "conteúdo qualquer")
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/documents", app.handleUpload())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error != "unsupported_format" {
		t.Errorf("error = %s, want unsupported_format", resp.Error)
	}
	if len(resp.SupportedFormats) == 0 {
		t.Error("SupportedFormats deveria listar os formatos aceitos")
	}
}

func TestUploadMissingFile(t *testing.T) {
	app, _, _ := newTestApp()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=test")
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/documents", app.handleUpload())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// =============================================================================
// Testes de Listagem (GET /api/v1/documents)
// =============================================================================

func TestListDocumentsEmpty(t *testing.T) {
	app, _, _ := newTestApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/documents", app.handleListDocuments())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp listResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
	if resp.Documents == nil {
		t.Error("documents não deveria ser nil (deve ser array vazio)")
	}
}

func TestListDocumentsWithData(t *testing.T) {
	app, store, _ := newTestApp()

	store.docs["doc-1"] = &storage.DocumentMeta{
		ID: "doc-1", Name: "manual.txt", FileType: "txt",
		FileSize: 1024, ChunkCount: 5, Status: "ready",
	}
	store.docs["doc-2"] = &storage.DocumentMeta{
		ID: "doc-2", Name: "faq.md", FileType: "md",
		FileSize: 2048, ChunkCount: 3, Status: "ready",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/documents", app.handleListDocuments())
	mux.ServeHTTP(rec, req)

	var resp listResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Total)
	}
	if resp.TotalChunks != 8 {
		t.Errorf("total_chunks = %d, want 8", resp.TotalChunks)
	}
}

// =============================================================================
// Testes de Detalhes (GET /api/v1/documents/{id})
// =============================================================================

func TestGetDocumentSuccess(t *testing.T) {
	app, store, _ := newTestApp()

	store.docs["abc-123"] = &storage.DocumentMeta{
		ID: "abc-123", Name: "report.pdf", FileType: "pdf",
		FileSize: 5120, ChunkCount: 10, Status: "ready",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/abc-123", nil)
	req.SetPathValue("id", "abc-123")
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/documents/{id}", app.handleGetDocument())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var doc storage.DocumentMeta
	json.NewDecoder(rec.Body).Decode(&doc)

	if doc.ID != "abc-123" {
		t.Errorf("ID = %s, want abc-123", doc.ID)
	}
	if doc.Name != "report.pdf" {
		t.Errorf("Name = %s, want report.pdf", doc.Name)
	}
}

func TestGetDocumentNotFound(t *testing.T) {
	app, _, _ := newTestApp()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/xyz-999", nil)
	req.SetPathValue("id", "xyz-999")
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/documents/{id}", app.handleGetDocument())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error != "not_found" {
		t.Errorf("error = %s, want not_found", resp.Error)
	}
}

// =============================================================================
// Testes de Remoção (DELETE /api/v1/documents/{id})
// =============================================================================

func TestDeleteDocumentSuccess(t *testing.T) {
	app, store, vs := newTestApp()

	store.docs["del-1"] = &storage.DocumentMeta{
		ID: "del-1", Name: "old-manual.txt", FileType: "txt",
		FileSize: 512, ChunkCount: 3, Status: "ready",
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/del-1", nil)
	req.SetPathValue("id", "del-1")
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/documents/{id}", app.handleDeleteDocument())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp deleteResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if !strings.Contains(resp.Message, "old-manual.txt") {
		t.Errorf("Message deveria conter o nome do documento: %s", resp.Message)
	}
	if !strings.Contains(resp.Message, "3 chunks deletados") {
		t.Errorf("Message deveria conter chunk count: %s", resp.Message)
	}

	// RN-21: Verifica que chunks foram removidos do VectorStore.
	if len(vs.deletedDocIDs) != 1 || vs.deletedDocIDs[0] != "del-1" {
		t.Errorf("VectorStore deveria ter recebido delete para del-1")
	}

	// Verifica remoção do storage.
	if _, ok := store.docs["del-1"]; ok {
		t.Error("Documento deveria ter sido removido do storage")
	}
}

func TestDeleteDocumentNotFound(t *testing.T) {
	app, _, _ := newTestApp()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/documents/nope", nil)
	req.SetPathValue("id", "nope")
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/documents/{id}", app.handleDeleteDocument())
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// =============================================================================
// Teste de Middleware
// =============================================================================

func TestRecoveryMiddleware(t *testing.T) {
	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("unexpected error")
	})

	handler := withRecovery(panickingHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	var resp errorResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Error != "internal_error" {
		t.Errorf("error = %s, want internal_error", resp.Error)
	}
}

func TestMaxFileSizeMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ok": "true"})
	})

	handler := withMaxFileSize(100, inner)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 200)))
	req.ContentLength = 200
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
}

func TestRequestLoggerMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	handler := withRequestLogger(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

// =============================================================================
// Teste de integração: upload + list + get + delete (fluxo completo)
// =============================================================================

func TestFullDocumentLifecycle(t *testing.T) {
	app, _, _ := newTestApp()

	mux := http.NewServeMux()
	registerRoutes(mux, app)

	// 1. Upload de documento.
	req := createMultipartRequest(t, "lifecycle.txt", "Conteúdo para teste de ciclo completo")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("upload: status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}

	var uploadResp uploadResponse
	json.NewDecoder(rec.Body).Decode(&uploadResp)
	docID := uploadResp.ID

	// 2. Listar: deve ter 1 documento.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var listResp listResponse
	json.NewDecoder(rec.Body).Decode(&listResp)

	if listResp.Total != 1 {
		t.Fatalf("list: total = %d, want 1", listResp.Total)
	}

	// 3. Get: detalhes do documento.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/documents/"+docID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get: status = %d, want 200", rec.Code)
	}

	// 4. Delete.
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/documents/"+docID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("delete: status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	// 5. List novamente: deve estar vazio.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/documents", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	json.NewDecoder(rec.Body).Decode(&listResp)
	if listResp.Total != 0 {
		t.Fatalf("list after delete: total = %d, want 0", listResp.Total)
	}
}
