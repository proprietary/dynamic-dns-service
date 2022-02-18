package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	do()
}

func do() {
	domainName, found := os.LookupEnv("DOMAIN_NAME")
	if !found {
		log.Fatal("Missing environment variable DOMAIN_NAME")
	}
	ctx := context.Background()
	ipv6, err := publicIpv6(ctx)
	if err != nil {
		log.Fatal(err)
	}
	ipv4, err := publicIpv4(ctx)
	if err != nil {
		log.Fatal(err)
	}
	cfdns := NewCloudflareDNS()
	dnsRecord, err := cfdns.CurrentDNSRecord(ctx, "A", domainName)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("DNS %s A record contains: %s\n", dnsRecord.Name(), dnsRecord.Content())
	if dnsRecord.Content() != ipv4 {
		log.Printf("DNS %s A record content (%s) differs from host IPv4 (%s). Updating...\n", dnsRecord.Name(), dnsRecord.Content(), ipv4)
		if err := cfdns.UpdateDNSRecord(ctx, dnsRecord.Identifier(), "A", domainName, ipv4, DEFAULT_TTL); err != nil {
			log.Fatal(err)
		}
		log.Printf("Updated A record of %s to %s\n", domainName, ipv4)
	}
	dnsRecord, err = cfdns.CurrentDNSRecord(ctx, "AAAA", domainName)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("DNS %s AAAA record contains: %s\n", dnsRecord.Name(), dnsRecord.Content())
	if dnsRecord.Content() != ipv6 {
		log.Printf("DNS %s AAAA record content (%s) differs from host IPv6 (%s). Updating...\n", dnsRecord.Name(), dnsRecord.Content(), ipv6)
		if err := cfdns.UpdateDNSRecord(ctx, dnsRecord.Identifier(), "AAAA", domainName, ipv6, DEFAULT_TTL); err != nil {
			log.Fatal(err)
		}
		log.Printf("Updated AAAA record of %s to %s\n", domainName, ipv6)
	}
}

const DEFAULT_TTL int = 60
const MS_TIMEOUT int = 10000

func publicIpv6(ctx context.Context) (string, error) {
	resolver := &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(MS_TIMEOUT),
			}
			return d.DialContext(ctx, "udp6", "resolver1.opendns.com:53")
		},
	}
	names, err := resolver.LookupIP(ctx, "ip6", "myip.opendns.com")
	if err != nil {
		return "", err
	}
	log.Println(names)
	return names[0].String(), nil
}

func publicIpv4(ctx context.Context) (string, error) {
	resolver := &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(MS_TIMEOUT),
			}
			return d.DialContext(ctx, "udp4", "resolver1.opendns.com:53")
		},
	}
	names, err := resolver.LookupIP(ctx, "ip4", "myip.opendns.com")
	if err != nil {
		return "", err
	}
	log.Println(names)
	return names[0].String(), nil
}

type CloudflareDNS struct {
	/// Zone ID from from Cloudflare dashboard Overview of website
	zoneId string
	/// Account ID from Cloudflare dashboard Overview of website
	accountId string
	/// API Token generated from the User Profile 'API Tokens' page: https://dash.cloudflare.com/profile/api-tokens
	apiToken string
}

type CloudflareDNSRecord struct {
	id         string
	name       string
	recordType string
	content    string
}

type DNSRecord interface {
	Name() string
	RecordType() string
	Content() string
	Identifier() string
}

func NewCloudflareDNS() CloudflareDNS {
	zoneId, found := os.LookupEnv("CF_ZONE_ID")
	if !found {
		log.Fatal("Missing environment variable: CF_ZONE_ID")
	}
	accountId, found := os.LookupEnv("CF_ACCOUNT_ID")
	if !found {
		log.Fatal("Missing environment variable: CF_ACCOUNT_ID")
	}
	apiToken, found := os.LookupEnv("CF_API_TOKEN")
	if !found {
		log.Fatal("Missing environment variable: CF_API_TOKEN")
	}
	return CloudflareDNS{
		zoneId:    zoneId,
		accountId: accountId,
		apiToken:  apiToken,
	}
}

func (cd *CloudflareDNS) CurrentDNSRecord(ctx context.Context, recordType, name string) (DNSRecord, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?page=1&per_page=100&type=%s&name=%s", cd.zoneId, recordType, name), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cd.apiToken))
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type CloudflareResponse struct {
		Success  bool     `json:"success"`
		Errors   []string `json:"errors"`
		Messages []string `json:"messages"`
		Result   []struct {
			Id         string      `json:"id"`
			Type       string      `json:"type"`
			Name       string      `json:"name"`
			Content    string      `json:"content"`
			Proxiable  bool        `json:"proxiable"`
			Proxied    bool        `json:"proxied"`
			Ttl        int         `json:"ttl"`
			Locked     bool        `json:"locked"`
			ZoneId     string      `json:"zone_id"`
			ZoneName   string      `json:"zone_name"`
			CreatedOn  string      `json:"created_on"`
			ModifiedOn string      `json:"modified_on"`
			Meta       interface{} `json:"meta"`
		} `json:"result"`
	}
	var cfr CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfr); err != nil {
		return nil, err
	}
	if cfr.Success == false {
		if len(cfr.Errors) > 0 {
			for _, errorMessage := range cfr.Errors {
				log.Fatalf("Error: %s\n", errorMessage)
			}
		}
		if len(cfr.Messages) > 0 {
			for _, message := range cfr.Messages {
				log.Printf("Message: %s\n", message)
			}
		}
		return nil, fmt.Errorf("Unsuccessful API call to retrieve DNS record")
	}
	if len(cfr.Result) > 1 {
		log.Println("Found more than one matching record; this shouldn't happen!")
	}
	result := cfr.Result[0]
	return &CloudflareDNSRecord{
		id:         result.Id,
		name:       result.Name,
		content:    result.Content,
		recordType: result.Type,
	}, nil
}

func (cd *CloudflareDNS) UpdateDNSRecord(ctx context.Context, identifier, recordType, name, content string, ttl int) error {
	dnsPatch := struct {
		Type    string `json:"type"`
		Name    string `json:"name"`
		Content string `json:"content"`
		Ttl     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}{
		Type:    recordType,
		Name:    name,
		Content: content,
		Ttl:     ttl,
		Proxied: false,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&dnsPatch); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", cd.zoneId, identifier), &buf)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cd.apiToken))
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	type CloudflareResponse struct {
		Success  bool        `json:"success"`
		Errors   []string    `json:"errors"`
		Messages []string    `json:"messages"`
		Result   interface{} `json:"result"`
	}
	var cfr CloudflareResponse

	if err := json.NewDecoder(resp.Body).Decode(&cfr); err != nil {
		return err
	}
	if cfr.Success == false {
		for _, errorMessage := range cfr.Errors {
			log.Printf("Error: %s\n", errorMessage)
		}
		for _, message := range cfr.Messages {
			log.Printf("Message: %s\n", message)
		}
		return fmt.Errorf("Unsuccessful API call to update DNS record")
	}
	return nil
}

func (cdr *CloudflareDNSRecord) Name() string {
	return cdr.name
}

func (cdr *CloudflareDNSRecord) RecordType() string {
	return cdr.recordType
}

func (cdr *CloudflareDNSRecord) Content() string {
	return cdr.content
}

func (cdr *CloudflareDNSRecord) Identifier() string {
	return cdr.id
}
