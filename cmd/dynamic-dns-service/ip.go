package main

import (
	"context"
	"flag"
	"net"
	"time"
)

var timeoutMillis = flag.Int("timeout_millis", 10000, "Timeout in milliseconds for DNS queries")

func publicIpv6(ctx context.Context) (string, error) {
	resolver := &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(*timeoutMillis),
			}
			return d.DialContext(ctx, "udp6", "resolver1.opendns.com:53")
		},
	}
	names, err := resolver.LookupIP(ctx, "ip6", "myip.opendns.com")
	if err != nil {
		return "", err
	}
	return names[0].String(), nil
}

func publicIpv4(ctx context.Context) (string, error) {
	resolver := &net.Resolver{
		PreferGo:     true,
		StrictErrors: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(*timeoutMillis),
			}
			return d.DialContext(ctx, "udp4", "resolver1.opendns.com:53")
		},
	}
	names, err := resolver.LookupIP(ctx, "ip4", "myip.opendns.com")
	if err != nil {
		return "", err
	}
	return names[0].String(), nil
}
