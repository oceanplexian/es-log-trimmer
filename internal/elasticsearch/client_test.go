package elasticsearch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/company/log-trimmer/internal/config"
	"github.com/company/log-trimmer/internal/logger"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		ESHost:  "https://localhost:9200",
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	if client == nil {
		t.Fatal("Expected client to be created")
	}
	if client.BaseURL != cfg.ESHost {
		t.Errorf("Expected BaseURL %s, got %s", cfg.ESHost, client.BaseURL)
	}
}

func TestGetClusterHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ClusterInfo{
			ClusterName: "test-cluster",
			Status:      "green",
			NodeCount:   3,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{ESHost: server.URL, SkipTLS: true}
	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	info, err := client.GetClusterHealth()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if info.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got %s", info.ClusterName)
	}
}

func TestDeleteIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"acknowledged": true}`))
	}))
	defer server.Close()

	cfg := &config.Config{ESHost: server.URL, SkipTLS: true}
	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	err := client.DeleteIndex("test-index")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestParseESSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1kb", 1024, false},
		{"1mb", 1024 * 1024, false},
		{"1gb", 1024 * 1024 * 1024, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		result, err := parseESSize(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("Expected error for input '%s'", tt.input)
		}
		if !tt.wantErr && result != tt.expected {
			t.Errorf("For input '%s', expected %d, got %d", tt.input, tt.expected, result)
		}
	}
}
