package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNewCloudflareDNS(t *testing.T) {
	origZoneID, origAccountID, origAPIToken := os.Getenv("CF_ZONE_ID"), os.Getenv("CF_ACCOUNT_ID"), os.Getenv("CF_API_TOKEN")
	defer func() {
		os.Setenv("CF_ZONE_ID", origZoneID)
		os.Setenv("CF_ACCOUNT_ID", origAccountID)
		os.Setenv("CF_API_TOKEN", origAPIToken)
	}()

	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "All env vars set",
			envVars: map[string]string{
				"CF_ZONE_ID":    "zone123",
				"CF_ACCOUNT_ID": "acc123",
				"CF_API_TOKEN":  "token123",
			},
			expectError: false,
		},
		{
			name:        "Missing env vars",
			envVars:     map[string]string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("CF_ZONE_ID")
			os.Unsetenv("CF_ACCOUNT_ID")
			os.Unsetenv("CF_API_TOKEN")

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cf, err := NewCloudflareDNS(nil)
			if tt.expectError {
				if err == nil {
					t.Error("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if cf == nil {
					t.Error("Expected CloudflareDNS instance, but got nil")
				}
			}
		})
	}
}

func TestCurrentDNSRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer testtoken" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		if !strings.Contains(r.URL.Path, "/zones/zone123/dns_records") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		response := map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"id":      "record123",
					"type":    "A",
					"name":    "example.com",
					"content": "192.0.2.1",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cf := &CloudflareDNS{
		zoneID:    "zone123",
		accountID: "acc123",
		apiToken:  "testtoken",
		client:    server.Client(),
		baseURL:   server.URL,
	}

	record, err := cf.CurrentDNSRecord(context.Background(), "A", "example.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if record.Identifier() != "record123" {
		t.Errorf("Expected record ID 'record123', got '%s'", record.Identifier())
	}
	if record.RecordType() != "A" {
		t.Errorf("Expected record type 'A', got '%s'", record.RecordType())
	}
	if record.Name() != "example.com" {
		t.Errorf("Expected record name 'example.com', got '%s'", record.Name())
	}
	if record.Content() != "192.0.2.1" {
		t.Errorf("Expected record content '192.0.2.1', got '%s'", record.Content())
	}
}

func TestUpdateDNSRecord(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH request, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer testtoken" {
			t.Errorf("Expected Authorization header, got %s", r.Header.Get("Authorization"))
		}
		if !strings.Contains(r.URL.Path, "/zones/zone123/dns_records/record123") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		expectedBody := map[string]interface{}{
			"type":    "A",
			"name":    "example.com",
			"content": "192.0.2.2",
			"ttl":     float64(120), // JSON numbers are floats
			"proxied": false,
		}

		for k, v := range expectedBody {
			if requestBody[k] != v {
				t.Errorf("Expected %s to be %v, got %v", k, v, requestBody[k])
			}
		}

		response := map[string]interface{}{
			"success": true,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cf := &CloudflareDNS{
		zoneID:    "zone123",
		accountID: "acc123",
		apiToken:  "testtoken",
		client:    server.Client(),
		baseURL:   server.URL,
	}

	err := cf.UpdateDNSRecord(context.Background(), "record123", "A", "example.com", "192.0.2.2", 120)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
