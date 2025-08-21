package elasticsearch

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/company/log-trimmer/internal/config"
	"github.com/company/log-trimmer/internal/logger"
)

// IndexInfo represents metadata about an Elasticsearch index
type IndexInfo struct {
	Name         string    `json:"index"`
	Health       string    `json:"health"`
	Status       string    `json:"status"`
	UUID         string    `json:"uuid"`
	Primary      string    `json:"pri"`
	Replica      string    `json:"rep"`
	DocsCount    int64     `json:"docs.count,string"`
	DocsDeleted  int64     `json:"docs.deleted,string"`
	StoreSize    string    `json:"store.size"`
	PrimarySize  string    `json:"pri.store.size"`
	SizeBytes    int64     // Calculated from StoreSize
	CreationDate time.Time // Calculated from index metadata
}

// ClusterInfo represents overall cluster information
type ClusterInfo struct {
	ClusterName string `json:"cluster_name"`
	Status      string `json:"status"`
	NodeCount   int    `json:"number_of_nodes"`
}

// Client wraps HTTP client for Elasticsearch operations
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Config     *config.Config
	Logger     *logger.Logger
}

// NewClient creates a new Elasticsearch client
func NewClient(cfg *config.Config, log *logger.Logger) *Client {
	tr := &http.Transport{}
	if cfg.SkipTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		BaseURL: cfg.ESHost,
		HTTPClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
		Config: cfg,
		Logger: log,
	}
}

// makeRequest makes an HTTP request to Elasticsearch
func (c *Client) makeRequest(method, path string) (*http.Response, error) {
	url := c.BaseURL + path

	c.Logger.Debug("elasticsearch", "request", "Making request", map[string]interface{}{
		"method": method,
		"url":    url,
	})

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.Config.Username != "" && c.Config.Password != "" {
		req.SetBasicAuth(c.Config.Username, c.Config.Password)
		c.Logger.Debug("elasticsearch", "auth", "Using basic authentication", map[string]interface{}{
			"username": c.Config.Username,
		})
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	c.Logger.Debug("elasticsearch", "response", "Received response", map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
	})

	return resp, nil
}

// GetClusterHealth retrieves cluster health information
func (c *Client) GetClusterHealth() (*ClusterInfo, error) {
	c.Logger.Info("elasticsearch", "cluster_health", "Retrieving cluster health information")

	resp, err := c.makeRequest("GET", "/_cluster/health")
	if err != nil {
		c.Logger.Error("elasticsearch", "cluster_health", "Failed to get cluster health", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err := fmt.Errorf("cluster health check failed with status %d", resp.StatusCode)
		c.Logger.Error("elasticsearch", "cluster_health", "Cluster health check failed", err, map[string]interface{}{
			"status_code": resp.StatusCode,
		})
		return nil, err
	}

	var clusterInfo ClusterInfo
	if err := json.NewDecoder(resp.Body).Decode(&clusterInfo); err != nil {
		c.Logger.Error("elasticsearch", "cluster_health", "Failed to decode cluster health response", err)
		return nil, fmt.Errorf("failed to decode cluster health response: %w", err)
	}

	c.Logger.Success("elasticsearch", "cluster_health", "Successfully retrieved cluster health", map[string]interface{}{
		"cluster_name": clusterInfo.ClusterName,
		"status":       clusterInfo.Status,
		"node_count":   clusterInfo.NodeCount,
	})

	return &clusterInfo, nil
}

// GetIndexes retrieves indexes matching the given pattern
func (c *Client) GetIndexes(pattern string) ([]IndexInfo, error) {
	c.Logger.Info("elasticsearch", "get_indexes", "Retrieving indexes", map[string]interface{}{
		"pattern": pattern,
	})

	path := fmt.Sprintf("/_cat/indices/%s?format=json&bytes=b", pattern)
	resp, err := c.makeRequest("GET", path)
	if err != nil {
		c.Logger.Error("elasticsearch", "get_indexes", "Failed to get indexes", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err := fmt.Errorf("failed to get indexes with status %d", resp.StatusCode)
		c.Logger.Error("elasticsearch", "get_indexes", "Get indexes request failed", err, map[string]interface{}{
			"status_code": resp.StatusCode,
		})
		return nil, err
	}

	var indexes []IndexInfo
	if err := json.NewDecoder(resp.Body).Decode(&indexes); err != nil {
		c.Logger.Error("elasticsearch", "get_indexes", "Failed to decode indexes response", err)
		return nil, fmt.Errorf("failed to decode indexes response: %w", err)
	}

	c.Logger.Info("elasticsearch", "get_indexes", "Found indexes", map[string]interface{}{
		"count":   len(indexes),
		"pattern": pattern,
	})

	// Enrich index information
	for i := range indexes {
		if err := c.enrichIndexInfo(&indexes[i]); err != nil {
			c.Logger.Warn("elasticsearch", "enrich_index", "Could not get creation date for index", map[string]interface{}{
				"index": indexes[i].Name,
				"error": err.Error(),
			})
		}
	}

	return indexes, nil
}

// enrichIndexInfo adds computed fields to index information
func (c *Client) enrichIndexInfo(index *IndexInfo) error {
	// Parse size from string format to bytes
	if sizeBytes, err := strconv.ParseInt(index.StoreSize, 10, 64); err == nil {
		index.SizeBytes = sizeBytes
	} else {
		// Fallback to parsing human-readable format
		if sizeBytes, err := parseESSize(index.StoreSize); err == nil {
			index.SizeBytes = sizeBytes
		}
	}

	// Get index settings to determine creation date
	path := fmt.Sprintf("/%s/_settings", index.Name)
	resp, err := c.makeRequest("GET", path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to get index settings with status %d", resp.StatusCode)
	}

	var settings map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return err
	}

	// Extract creation date
	if indexSettings, ok := settings[index.Name].(map[string]interface{}); ok {
		if settingsObj, ok := indexSettings["settings"].(map[string]interface{}); ok {
			if indexObj, ok := settingsObj["index"].(map[string]interface{}); ok {
				if creationDateStr, ok := indexObj["creation_date"].(string); ok {
					if creationTimestamp, err := strconv.ParseInt(creationDateStr, 10, 64); err == nil {
						index.CreationDate = time.Unix(0, creationTimestamp*int64(time.Millisecond))
					}
				}
			}
		}
	}

	return nil
}

// DeleteIndex deletes the specified index
func (c *Client) DeleteIndex(indexName string) error {
	c.Logger.Info("elasticsearch", "delete_index", "Deleting index", map[string]interface{}{
		"index": indexName,
	})

	path := fmt.Sprintf("/%s", indexName)
	resp, err := c.makeRequest("DELETE", path)
	if err != nil {
		c.Logger.Error("elasticsearch", "delete_index", "Failed to delete index", err, map[string]interface{}{
			"index": indexName,
		})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to delete index with status %d: %s", resp.StatusCode, string(body))
		c.Logger.Error("elasticsearch", "delete_index", "Delete index request failed", err, map[string]interface{}{
			"index":       indexName,
			"status_code": resp.StatusCode,
			"response":    string(body),
		})
		return err
	}

	c.Logger.Success("elasticsearch", "delete_index", "Successfully deleted index", map[string]interface{}{
		"index": indexName,
	})

	return nil
}

// AnalyzeIndexes analyzes indexes and determines which ones should be deleted
func (c *Client) AnalyzeIndexes(indexes []IndexInfo) ([]IndexInfo, AnalysisResult) {
	c.Logger.Info("analysis", "analyze_indexes", "Analyzing indexes for deletion", map[string]interface{}{
		"total_indexes": len(indexes),
	})

	// Sort indexes by creation date (oldest first) for deletion prioritization
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].CreationDate.Before(indexes[j].CreationDate)
	})

	var toDelete []IndexInfo
	var totalSize int64

	// Calculate current total size
	for _, index := range indexes {
		totalSize += index.SizeBytes
	}

	result := AnalysisResult{
		TotalIndexes: len(indexes),
		TotalSize:    totalSize,
		ToDelete:     0,
		DeletedSize:  0,
	}

	c.Logger.Info("analysis", "current_state", "Current cluster state", map[string]interface{}{
		"total_indexes": len(indexes),
		"total_size":    totalSize,
	})

	// Apply age filter first
	if c.Config.MaxAgeDuration > 0 {
		cutoffTime := time.Now().Add(-c.Config.MaxAgeDuration)
		for _, index := range indexes {
			if index.CreationDate.Before(cutoffTime) {
				toDelete = append(toDelete, index)
				result.DeletedSize += index.SizeBytes
			}
		}
		c.Logger.Info("analysis", "age_filter", "Applied age filter", map[string]interface{}{
			"max_age":     c.Config.MaxAge,
			"cutoff_time": cutoffTime,
			"age_deletes": len(toDelete),
		})
	}

	// Apply size filter
	if c.Config.MaxSizeBytes > 0 && totalSize > c.Config.MaxSizeBytes {
		excessSize := totalSize - c.Config.MaxSizeBytes
		c.Logger.Warn("analysis", "size_filter", "Total size exceeds limit", map[string]interface{}{
			"total_size":  totalSize,
			"max_size":    c.Config.MaxSizeBytes,
			"excess_size": excessSize,
		})

		// Calculate how much we're already deleting from age filter
		deletedSize := result.DeletedSize

		for _, index := range indexes {
			// Skip if already marked for deletion by age
			alreadyMarked := false
			for _, deleted := range toDelete {
				if deleted.Name == index.Name {
					alreadyMarked = true
					break
				}
			}

			if !alreadyMarked && (deletedSize < excessSize) {
				toDelete = append(toDelete, index)
				deletedSize += index.SizeBytes
			}
		}

		result.DeletedSize = deletedSize
	}

	result.ToDelete = len(toDelete)

	c.Logger.Info("analysis", "result", "Analysis complete", map[string]interface{}{
		"total_indexes":     result.TotalIndexes,
		"indexes_to_delete": result.ToDelete,
		"size_to_delete":    result.DeletedSize,
	})

	return toDelete, result
}

// AnalysisResult contains the results of index analysis
type AnalysisResult struct {
	TotalIndexes int   `json:"total_indexes"`
	TotalSize    int64 `json:"total_size"`
	ToDelete     int   `json:"to_delete"`
	DeletedSize  int64 `json:"deleted_size"`
}

// parseESSize parses Elasticsearch size format
func parseESSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	// Handle Elasticsearch size formats like "4.9gb", "123.4mb", etc.
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([kmgt]?b)$`)
	matches := re.FindStringSubmatch(strings.ToLower(sizeStr))
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid ES size format: %s", sizeStr)
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	unit := matches[2]
	multiplier := int64(1)

	switch unit {
	case "b":
		multiplier = 1
	case "kb":
		multiplier = 1024
	case "mb":
		multiplier = 1024 * 1024
	case "gb":
		multiplier = 1024 * 1024 * 1024
	case "tb":
		multiplier = 1024 * 1024 * 1024 * 1024
	}

	return int64(value * float64(multiplier)), nil
}
