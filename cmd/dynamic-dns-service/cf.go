package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type CloudflareDNS struct {
	zoneID    string
	accountID string
	apiToken  string
	client    *http.Client
	baseURL   string
}

type DNSRecord interface {
	Name() string
	RecordType() string
	Content() string
	Identifier() string
}

type CloudflareDNSRecord struct {
	id         string
	name       string
	recordType string
	content    string
}

type CloudflareErrors []struct {
	Message string `json:"message"`
}

func NewCloudflareDNS(client *http.Client) (*CloudflareDNS, error) {
	zoneID := os.Getenv("CF_ZONE_ID")
	accountID := os.Getenv("CF_ACCOUNT_ID")
	apiToken := os.Getenv("CF_API_TOKEN")

	if zoneID == "" || accountID == "" || apiToken == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	if client == nil {
		client = http.DefaultClient
	}

	return &CloudflareDNS{
		zoneID:    zoneID,
		accountID: accountID,
		apiToken:  apiToken,
		client:    client,
		baseURL:   "https://api.cloudflare.com/client/v4",
	}, nil
}

func (cd *CloudflareDNS) CurrentDNSRecord(ctx context.Context, recordType, name string) (DNSRecord, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?page=1&per_page=100&type=%s&name=%s", cd.baseURL, cd.zoneID, recordType, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	cd.setHeaders(req)

	resp, err := cd.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return cd.readCurrentDNSRecord(resp.Body)
}

func (cd *CloudflareDNS) UpdateDNSRecord(ctx context.Context, identifier, recordType, name, content string, ttl int) error {
	dnsPatch := struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}{
		Type:    recordType,
		Name:    name,
		Content: content,
		TTL:     ttl,
		Proxied: false,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&dnsPatch); err != nil {
		return fmt.Errorf("failed to encode DNS patch: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", cd.baseURL, cd.zoneID, identifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	cd.setHeaders(req)

	resp, err := cd.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	return cd.handleResponse(resp.Body)
}

func (cd *CloudflareDNS) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+cd.apiToken)
	req.Header.Set("Content-Type", "application/json")
}

func (cd *CloudflareDNS) readCurrentDNSRecord(r io.Reader) (DNSRecord, error) {
	var cfr struct {
		Success  bool             `json:"success"`
		Errors   CloudflareErrors `json:"errors"`
		Messages []string         `json:"messages"`
		Result   []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Name    string `json:"name"`
			Content string `json:"content"`
		} `json:"result"`
	}

	if err := json.NewDecoder(r).Decode(&cfr); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !cfr.Success {
		return nil, cd.formatErrorMessage(cfr.Errors, cfr.Messages)
	}

	if len(cfr.Result) == 0 {
		return nil, fmt.Errorf("no matching DNS record found")
	}

	if len(cfr.Result) > 1 {
		return nil, fmt.Errorf("found more than one matching record")
	}

	result := cfr.Result[0]
	return &CloudflareDNSRecord{
		id:         result.ID,
		name:       result.Name,
		content:    result.Content,
		recordType: result.Type,
	}, nil
}

func (cd *CloudflareDNS) handleResponse(r io.Reader) error {
	var cfr struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
		Messages []string `json:"messages"`
	}

	if err := json.NewDecoder(r).Decode(&cfr); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !cfr.Success {
		return cd.formatErrorMessage(cfr.Errors, cfr.Messages)
	}

	return nil
}

func (cd *CloudflareDNS) formatErrorMessage(errors CloudflareErrors, messages []string) error {
	var errorMsg string
	for _, err := range errors {
		errorMsg += fmt.Sprintf("Error: %s\n", err.Message)
	}
	for _, msg := range messages {
		errorMsg += fmt.Sprintf("Message: %s\n", msg)
	}
	return fmt.Errorf("unsuccessful API call: %s", errorMsg)
}

func (cdr *CloudflareDNSRecord) Name() string       { return cdr.name }
func (cdr *CloudflareDNSRecord) RecordType() string { return cdr.recordType }
func (cdr *CloudflareDNSRecord) Content() string    { return cdr.content }
func (cdr *CloudflareDNSRecord) Identifier() string { return cdr.id }
