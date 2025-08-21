package elasticsearch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/company/log-trimmer/internal/config"
	"github.com/company/log-trimmer/internal/logger"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		ESHost:   "https://localhost:9200",
		Username: "admin",
		Password: "secret",
		SkipTLS:  true,
	}

	log, err := logger.New(logger.DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	client := NewClient(cfg, log)

	if client.BaseURL != cfg.ESHost {
		t.Errorf("Expected BaseURL %s, got %s", cfg.ESHost, client.BaseURL)
	}

	if client.Config != cfg {
		t.Errorf("Expected config to be set")
	}

	if client.Logger != log {
		t.Errorf("Expected logger to be set")
	}

	if client.HTTPClient == nil {
		t.Errorf("Expected HTTP client to be initialized")
	}

	if client.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.HTTPClient.Timeout)
	}
}

func TestGetClusterHealth(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_cluster/health" {
			t.Errorf("Expected path /_cluster/health, got %s", r.URL.Path)
		}

		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		response := ClusterInfo{
			ClusterName: "test-cluster",
			Status:      "green",
			NodeCount:   3,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.Config{
		ESHost:  server.URL,
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	clusterInfo, err := client.GetClusterHealth()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if clusterInfo.ClusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got %s", clusterInfo.ClusterName)
	}

	if clusterInfo.Status != "green" {
		t.Errorf("Expected status 'green', got %s", clusterInfo.Status)
	}

	if clusterInfo.NodeCount != 3 {
		t.Errorf("Expected node count 3, got %d", clusterInfo.NodeCount)
	}
}

func TestGetClusterHealthError(t *testing.T) {
	// Create test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.Config{
		ESHost:  server.URL,
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	_, err := client.GetClusterHealth()
	if err == nil {
		t.Errorf("Expected error for 500 response, got nil")
	}

	expectedMsg := "cluster health check failed with status 500"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got %s", expectedMsg, err.Error())
	}
}

func TestGetIndexes(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle both index listing and settings requests
		if strings.Contains(r.URL.Path, "/_cat/indices/logs-*") {
			// Check for format=json and bytes=b parameters
			if !strings.Contains(r.URL.RawQuery, "format=json") {
				t.Errorf("Expected format=json parameter in query")
			}

			if !strings.Contains(r.URL.RawQuery, "bytes=b") {
				t.Errorf("Expected bytes=b parameter in query")
			}
		} else if strings.Contains(r.URL.Path, "/_settings") {
			// Handle settings requests for individual indexes
			indexName := strings.Split(r.URL.Path, "/")[1]
			settings := map[string]interface{}{
				indexName: map[string]interface{}{
					"settings": map[string]interface{}{
						"index": map[string]interface{}{
							"creation_date": "1692633600000", // Mock timestamp
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(settings)
			return
		}

		indexes := []IndexInfo{
			{
				Name:        "logs-2025.08.20",
				Health:      "green",
				Status:      "open",
				UUID:        "abc123",
				Primary:     "1",
				Replica:     "0",
				DocsCount:   1000,
				DocsDeleted: 0,
				StoreSize:   "1048576", // 1MB in bytes
				PrimarySize: "1048576",
			},
			{
				Name:        "logs-2025.08.21",
				Health:      "yellow",
				Status:      "open",
				UUID:        "def456",
				Primary:     "1",
				Replica:     "0",
				DocsCount:   2000,
				DocsDeleted: 0,
				StoreSize:   "2097152", // 2MB in bytes
				PrimarySize: "2097152",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(indexes)
	}))
	defer server.Close()

	// Mock the settings endpoint for creation dates
	settingsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		indexName := strings.Trim(strings.Split(r.URL.Path, "/")[1], "/")

		settings := map[string]interface{}{
			indexName: map[string]interface{}{
				"settings": map[string]interface{}{
					"index": map[string]interface{}{
						"creation_date": "1692633600000", // Mock timestamp
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	}))
	defer settingsServer.Close()

	cfg := &config.Config{
		ESHost:  server.URL,
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	// Override the base URL for settings requests by creating a custom client
	// For testing purposes, we'll just test the main functionality
	indexes, err := client.GetIndexes("logs-*")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(indexes) != 2 {
		t.Errorf("Expected 2 indexes, got %d", len(indexes))
	}

	// Check first index
	if indexes[0].Name != "logs-2025.08.20" {
		t.Errorf("Expected first index name 'logs-2025.08.20', got %s", indexes[0].Name)
	}

	if indexes[0].Health != "green" {
		t.Errorf("Expected first index health 'green', got %s", indexes[0].Health)
	}

	if indexes[0].DocsCount != 1000 {
		t.Errorf("Expected first index docs count 1000, got %d", indexes[0].DocsCount)
	}

	if indexes[0].SizeBytes != 1048576 {
		t.Errorf("Expected first index size 1048576 bytes, got %d", indexes[0].SizeBytes)
	}

	// Check second index
	if indexes[1].Name != "logs-2025.08.21" {
		t.Errorf("Expected second index name 'logs-2025.08.21', got %s", indexes[1].Name)
	}

	if indexes[1].Health != "yellow" {
		t.Errorf("Expected second index health 'yellow', got %s", indexes[1].Health)
	}

	if indexes[1].DocsCount != 2000 {
		t.Errorf("Expected second index docs count 2000, got %d", indexes[1].DocsCount)
	}

	if indexes[1].SizeBytes != 2097152 {
		t.Errorf("Expected second index size 2097152 bytes, got %d", indexes[1].SizeBytes)
	}
}

func TestDeleteIndex(t *testing.T) {
	indexToDelete := "test-index"
	deleteCalled := false

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/" + indexToDelete
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}

		deleteCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"acknowledged": true}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		ESHost:  server.URL,
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	err := client.DeleteIndex(indexToDelete)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !deleteCalled {
		t.Errorf("Expected delete endpoint to be called")
	}
}

func TestDeleteIndexError(t *testing.T) {
	// Create test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "index_not_found_exception"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		ESHost:  server.URL,
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	err := client.DeleteIndex("nonexistent-index")
	if err == nil {
		t.Errorf("Expected error for 404 response, got nil")
	}

	expectedMsg := "failed to delete index with status 404"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing '%s', got %s", expectedMsg, err.Error())
	}
}

func TestAnalyzeIndexes(t *testing.T) {
	cfg := &config.Config{
		MaxAgeDuration: 7 * 24 * time.Hour, // 7 days
		MaxSizeBytes:   5 * 1024 * 1024,    // 5MB
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	// Create test indexes with different ages and sizes
	now := time.Now()
	indexes := []IndexInfo{
		{
			Name:         "old-index",
			SizeBytes:    2 * 1024 * 1024,               // 2MB
			CreationDate: now.Add(-10 * 24 * time.Hour), // 10 days old
		},
		{
			Name:         "recent-index",
			SizeBytes:    1 * 1024 * 1024,              // 1MB
			CreationDate: now.Add(-2 * 24 * time.Hour), // 2 days old
		},
		{
			Name:         "large-index",
			SizeBytes:    4 * 1024 * 1024,              // 4MB
			CreationDate: now.Add(-1 * 24 * time.Hour), // 1 day old
		},
	}

	toDelete, result := client.AnalyzeIndexes(indexes)

	// Verify analysis result
	if result.TotalIndexes != 3 {
		t.Errorf("Expected TotalIndexes 3, got %d", result.TotalIndexes)
	}

	expectedTotalSize := int64(7 * 1024 * 1024) // 7MB total
	if result.TotalSize != expectedTotalSize {
		t.Errorf("Expected TotalSize %d, got %d", expectedTotalSize, result.TotalSize)
	}

	// Should delete old-index (age) and large-index (size limit exceeded)
	if len(toDelete) < 1 {
		t.Errorf("Expected at least 1 index to be deleted, got %d", len(toDelete))
	}

	// Check that old-index is marked for deletion (due to age)
	foundOldIndex := false
	for _, index := range toDelete {
		if index.Name == "old-index" {
			foundOldIndex = true
			break
		}
	}

	if !foundOldIndex {
		t.Errorf("Expected old-index to be marked for deletion due to age")
	}
}

func TestParseESSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1b", 1, false},
		{"1kb", 1024, false},
		{"1mb", 1024 * 1024, false},
		{"1gb", 1024 * 1024 * 1024, false},
		{"1tb", 1024 * 1024 * 1024 * 1024, false},
		{"1.5gb", int64(1.5 * 1024 * 1024 * 1024), false},
		{"500mb", 500 * 1024 * 1024, false},
		{"", 0, false},
		{"invalid", 0, true},
		{"100xb", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseESSize(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input '%s', got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input '%s': %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("For input '%s', expected %d, got %d", tt.input, tt.expected, result)
			}
		})
	}
}

func TestMakeRequestWithAuth(t *testing.T) {
	authCalled := false

	// Create test server that checks for auth
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Errorf("Expected basic auth to be present")
			return
		}

		if username != "testuser" || password != "testpass" {
			t.Errorf("Expected auth testuser:testpass, got %s:%s", username, password)
			return
		}

		authCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	cfg := &config.Config{
		ESHost:   server.URL,
		Username: "testuser",
		Password: "testpass",
		SkipTLS:  true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	resp, err := client.makeRequest("GET", "/test")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if !authCalled {
		t.Errorf("Expected auth to be checked")
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestEnrichIndexInfo(t *testing.T) {
	// Create test server for settings endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "_settings") {
			t.Errorf("Expected settings endpoint, got %s", r.URL.Path)
		}

		settings := map[string]interface{}{
			"test-index": map[string]interface{}{
				"settings": map[string]interface{}{
					"index": map[string]interface{}{
						"creation_date": "1692633600000", // Mock timestamp
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	}))
	defer server.Close()

	cfg := &config.Config{
		ESHost:  server.URL,
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	index := &IndexInfo{
		Name:      "test-index",
		StoreSize: "1048576", // 1MB in bytes
	}

	err := client.enrichIndexInfo(index)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if index.SizeBytes != 1048576 {
		t.Errorf("Expected size to be parsed to 1048576 bytes, got %d", index.SizeBytes)
	}

	if index.CreationDate.IsZero() {
		t.Errorf("Expected creation date to be set")
	}
}

// Benchmark tests
func BenchmarkParseESSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parseESSize("100gb")
	}
}

func BenchmarkAnalyzeIndexes(b *testing.B) {
	cfg := &config.Config{
		MaxAgeDuration: 7 * 24 * time.Hour,
		MaxSizeBytes:   100 * 1024 * 1024,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	// Create test data
	now := time.Now()
	indexes := make([]IndexInfo, 100)
	for i := 0; i < 100; i++ {
		indexes[i] = IndexInfo{
			Name:         fmt.Sprintf("index-%d", i),
			SizeBytes:    int64(i * 1024 * 1024), // Variable sizes
			CreationDate: now.Add(time.Duration(-i) * time.Hour),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.AnalyzeIndexes(indexes)
	}
}

// Test error handling
func TestClientErrorHandling(t *testing.T) {
	// Test with invalid URL
	cfg := &config.Config{
		ESHost:  "invalid://url",
		SkipTLS: true,
	}

	log, _ := logger.New(logger.DefaultConfig())
	client := NewClient(cfg, log)

	_, err := client.GetClusterHealth()
	if err == nil {
		t.Errorf("Expected error for invalid URL, got nil")
	}
}
