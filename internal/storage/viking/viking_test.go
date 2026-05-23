package viking

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newMockServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	srv := httptest.NewServer(handler)
	return srv, NewClient(srv.URL, "test-api-key")
}

func TestClient_AddResource(t *testing.T) {
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/resources" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("expected Bearer test-api-key, got %s", auth)
		}
		body, _ := io.ReadAll(r.Body)
		var req AddResourceRequest
		_ = json.Unmarshal(body, &req)
		if req.Path != "https://cursor.com" {
			t.Errorf("expected path https://cursor.com, got %s", req.Path)
		}
		resp := AddResourceResponse{RootURI: "viking://resources/cursor", TaskID: "task-123"}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	res, err := client.AddResource(context.Background(), &AddResourceRequest{
		Path: "https://cursor.com",
		To:   "viking://resources/cursor",
		Wait: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RootURI != "viking://resources/cursor" {
		t.Fatalf("expected RootURI viking://resources/cursor, got %s", res.RootURI)
	}
}

func TestClient_Find(t *testing.T) {
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/find" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		results := []FindResult{
			{URI: "viking://resources/cursor/README", Content: "Cursor is an AI editor", Level: "L1", Score: 0.95, SourceType: "web"},
		}
		_ = json.NewEncoder(w).Encode(results)
	})
	defer srv.Close()

	results, err := client.Find(context.Background(), &FindRequest{
		Query: "AI editor",
		Path:  "viking://resources/",
		Level: "L1",
		TopK:  5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Level != "L1" {
		t.Fatalf("expected L1, got %s", results[0].Level)
	}
}

func TestClient_Grep(t *testing.T) {
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/grep" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		results := []FindResult{
			{URI: "viking://resources/cursor/deprecated.md", Content: "deprecated feature X", Level: "L0", Score: 0.80},
		}
		_ = json.NewEncoder(w).Encode(results)
	})
	defer srv.Close()

	results, err := client.Grep(context.Background(), "deprecated", "viking://resources/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !strings.Contains(results[0].Content, "deprecated") {
		t.Fatalf("expected content to contain 'deprecated', got %s", results[0].Content)
	}
}

func TestClient_Glob(t *testing.T) {
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/glob" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		results := []string{
			"viking://resources/cursor/README.md",
			"viking://resources/cursor/CHANGELOG.md",
		}
		_ = json.NewEncoder(w).Encode(results)
	})
	defer srv.Close()

	results, err := client.Glob(context.Background(), "*.md", "viking://resources/cursor/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestClient_ErrorStatus(t *testing.T) {
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})
	defer srv.Close()

	_, err := client.AddResource(context.Background(), &AddResourceRequest{Path: "/x"})
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected error to contain 500, got %v", err)
	}
}

func TestSearchByLevel_InvalidLevel(t *testing.T) {
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {})
	defer srv.Close()

	_, err := client.SearchByLevel(context.Background(), "q", "p", "invalid", 5)
	if err == nil {
		t.Fatal("expected error for invalid level")
	}
}

func TestBatchGrep_Deduplicates(t *testing.T) {
	callCount := 0
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		results := []FindResult{
			{URI: "v1", Content: "a", Level: "L0", Score: 0.9},
			{URI: "v2", Content: "b", Level: "L0", Score: 0.8},
		}
		_ = json.NewEncoder(w).Encode(results)
	})
	defer srv.Close()

	results, err := client.BatchGrep(context.Background(), []string{"a", "b"}, "viking://")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Two patterns each return v1+v2, but dedup should leave 2 unique URIs.
	if len(results) != 2 {
		t.Fatalf("expected 2 deduplicated results, got %d", len(results))
	}
	if callCount != 2 {
		t.Fatalf("expected 2 Grep calls, got %d", callCount)
	}
}

func TestSearchForContradictions(t *testing.T) {
	srv, client := newMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/grep" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		_ = json.Unmarshal(body, &req)
		pattern := req["pattern"]

		var results []FindResult
		if strings.Contains(pattern, "not") {
			results = []FindResult{
				{URI: "viking://resources/cursor/deprecated.md", Content: "not supported", Level: "L0", Score: 0.85},
			}
		}
		_ = json.NewEncoder(w).Encode(results)
	})
	defer srv.Close()

	hits, err := client.SearchForContradictions(context.Background(), "best product", "viking://resources/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected contradictory evidence")
	}
}

func TestCompetifyVikingPaths(t *testing.T) {
	p := NewCompetifyPaths()
	cases := []struct {
		got  string
		want string
	}{
		{p.TaskPath("task-1"), "viking://competify/tasks/task-1/"},
		{p.TaskCollectorPath("task-1", "web"), "viking://competify/tasks/task-1/collectors/web/"},
		{p.TaskAnalyzerPath("task-1", "feature"), "viking://competify/tasks/task-1/analyzers/feature/"},
		{p.OntologyPath(), "viking://competify/ontology/"},
		{p.CompetitorOntologyPath("cursor"), "viking://competify/ontology/competitors/cursor/"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Fatalf("expected %s, got %s", tc.want, tc.got)
		}
	}
}
