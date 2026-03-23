package config

import (
	"fmt"
	"log/slog"
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
			slog.Warn("invalid proxy URL, falling back to direct", "component", "proxy", "url", proxyCfg.HTTPProxy, "error", err)
		} else {
			switch proxyURL.Scheme {
			case "http", "https":
				transport.Proxy = http.ProxyURL(proxyURL)
				slog.Info("using HTTP proxy", "component", "proxy", "url", proxyCfg.HTTPProxy)
			case "socks5", "socks5h":
				dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
				if err != nil {
					slog.Warn("failed to create SOCKS dialer, falling back to direct", "component", "proxy", "url", proxyCfg.HTTPProxy, "error", err)
				} else {
					// proxy.Dialer only provides Dial(network, addr), wrap it for DialContext
					transport.DialContext = nil
					transport.Dial = func(network, addr string) (net.Conn, error) {
						return dialer.Dial(network, addr)
					}
					slog.Info("using SOCKS proxy", "component", "proxy", "url", proxyCfg.HTTPProxy)
				}
			default:
				slog.Warn("unsupported proxy scheme, falling back to direct", "component", "proxy", "scheme", proxyURL.Scheme)
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
