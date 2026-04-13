package llamaparse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLlamaParser_Parse(t *testing.T) {
	// Mock LlamaParse API
	jobID := "test-job-id"
	mux := http.NewServeMux()

	// Upload
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Expected multipart/form-data, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": jobID})
	})

	// Status
	mux.HandleFunc("/job/"+jobID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "SUCCESS"})
	})

	// Result
	mux.HandleFunc("/job/"+jobID+"/result/markdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"markdown": "parsed content"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	parser := NewLlamaParser("test-api-key",
		WithBaseURL(server.URL),
		WithCheckInterval(10*time.Millisecond),
	)

	ctx := context.Background()
	reader := strings.NewReader("dummy content")
	docs, err := parser.Parse(ctx, reader, &ParseOptions{Filename: "test.pdf"})

	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("Expected 1 document, got %d", len(docs))
	}

	if docs[0].Content != "parsed content" {
		t.Errorf("Expected content 'parsed content', got '%s'", docs[0].Content)
	}

	if docs[0].MetaData["job_id"] != jobID {
		t.Errorf("Expected jobID '%s', got '%v'", jobID, docs[0].MetaData["job_id"])
	}
}

func TestLlamaParser_PollTimeout(t *testing.T) {
	jobID := "timeout-job"
	mux := http.NewServeMux()

	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": jobID})
	})

	mux.HandleFunc("/job/"+jobID, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "PENDING"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	parser := NewLlamaParser("test-api-key",
		WithBaseURL(server.URL),
		WithCheckInterval(10*time.Millisecond),
		WithMaxTimeout(50*time.Millisecond),
	)

	ctx := context.Background()
	reader := strings.NewReader("dummy content")
	_, err := parser.Parse(ctx, reader)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error message, got: %v", err)
	}
}
