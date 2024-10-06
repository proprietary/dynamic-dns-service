package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
)

var (
	domainName = flag.String("domain", "", "Domain name to update")
	dnsTtl     = flag.Int("ttl", 60, "TTL to set on the DNS records")
)

func main() {
	flag.Parse()
	if *domainName == "" {
		maybeDomainName, found := os.LookupEnv("DOMAIN_NAME")
		if !found {
			log.Fatal("Missing domain name, which can be set with the -domain flag or the DOMAIN_NAME environment variable")
		} else {
			domainName = &maybeDomainName
		}
	}
	ctx := context.Background()
	httpClient := &http.Client{}
	cfdns, err := NewCloudflareDNS(httpClient)
	if err != nil {
		log.Fatal(err)
	}
	if err := updateDNSRecord(ctx, cfdns, "A", *domainName); err != nil {
		log.Printf("Error updating A record: %s\n", err)
	}
	if err := updateDNSRecord(ctx, cfdns, "AAAA", *domainName); err != nil {
		log.Printf("Error updating AAAA record: %s\n", err)
	}
}

func updateDNSRecord(ctx context.Context, cfdns *CloudflareDNS, recordType string, domainName string) error {
	dnsRecord, err := cfdns.CurrentDNSRecord(ctx, recordType, domainName)
	if err != nil {
		return err
	}
	var newIp string
	switch recordType {
	case "A":
		newIp, err = publicIpv4(ctx)
		if err != nil {
			return err
		}
	case "AAAA":
		newIp, err = publicIpv6(ctx)
		if err != nil {
			return err
		}
	}
	if dnsRecord.Content() != newIp {
		log.Printf("DNS %s %s record content (%s) differs from host (%s). Updating...\n", dnsRecord.Name(), dnsRecord.RecordType(), dnsRecord.Content(), newIp)
		if err := cfdns.UpdateDNSRecord(ctx, dnsRecord.Identifier(), dnsRecord.RecordType(), dnsRecord.Name(), newIp, *dnsTtl); err != nil {
			return err
		}
		log.Printf("Updated %s record of %s to %s\n", dnsRecord.RecordType(), dnsRecord.Name(), newIp)
	}
	return nil
}
