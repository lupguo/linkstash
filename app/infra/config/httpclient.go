package config

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// NewHTTPClient creates an *http.Client that respects the proxy configuration.
// Supported schemes: http, https, socks5, socks5h.
// If no proxy is configured, a plain client with the given timeout is returned.
func NewHTTPClient(proxyCfg ProxyConfig, timeout time.Duration) *http.Client {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:  10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	if proxyCfg.HTTPProxy != "" {
		proxyURL, err := url.Parse(proxyCfg.HTTPProxy)
		if err != nil {
			log.Printf("[Proxy] invalid proxy URL %q: %v, falling back to direct", proxyCfg.HTTPProxy, err)
		} else {
			switch proxyURL.Scheme {
			case "http", "https":
				transport.Proxy = http.ProxyURL(proxyURL)
				log.Printf("[Proxy] using HTTP proxy: %s", proxyCfg.HTTPProxy)
			case "socks5", "socks5h":
				dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
				if err != nil {
					log.Printf("[Proxy] failed to create SOCKS dialer for %q: %v, falling back to direct", proxyCfg.HTTPProxy, err)
				} else {
					// proxy.Dialer only provides Dial(network, addr), wrap it for DialContext
					transport.DialContext = nil
					transport.Dial = func(network, addr string) (net.Conn, error) {
						return dialer.Dial(network, addr)
					}
					log.Printf("[Proxy] using SOCKS proxy: %s", proxyCfg.HTTPProxy)
				}
			default:
				log.Printf("[Proxy] unsupported proxy scheme %q, falling back to direct", proxyURL.Scheme)
			}
		}
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// ProxyDescription returns a human-readable description of the proxy setting.
func (p ProxyConfig) ProxyDescription() string {
	if p.HTTPProxy == "" {
		return "direct (no proxy)"
	}
	return fmt.Sprintf("proxy: %s", p.HTTPProxy)
}
