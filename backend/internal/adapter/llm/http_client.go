package llm

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

func newHTTPClient(socks5Proxy string) (*http.Client, error) {
	if socks5Proxy == "" {
		return &http.Client{}, nil
	}

	proxyURL, err := parseSOCKS5ProxyURL(socks5Proxy)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	transport.DialContext = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext

	return &http.Client{Transport: transport}, nil
}

func parseSOCKS5ProxyURL(raw string) (*url.URL, error) {
	proxyURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse SOCKS5 proxy URL: %w", err)
	}
	if proxyURL.Scheme != "socks5" && proxyURL.Scheme != "socks5h" {
		return nil, fmt.Errorf("unsupported SOCKS5 proxy scheme %q", proxyURL.Scheme)
	}
	if proxyURL.Host == "" {
		return nil, fmt.Errorf("SOCKS5 proxy host is required")
	}
	return proxyURL, nil
}
