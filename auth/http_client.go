// Package auth 提供认证相关功能的 HTTP 客户端
package auth

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// 全局 HTTP 客户端存储，支持运行时代理重配置
var httpClientStore atomic.Pointer[http.Client]

// authProxyClientCache caches per-proxy auth HTTP clients.
var authProxyClientCache sync.Map

// httpClient 返回当前全局 auth HTTP 客户端
func httpClient() *http.Client {
	return httpClientStore.Load()
}

func init() {
	InitHttpClient("")
}

// GetAuthClientForProxy returns an auth HTTP client for the given proxy URL.
// If proxyURL is empty, returns the global auth HTTP client.
func GetAuthClientForProxy(proxyURL string) *http.Client {
	if proxyURL == "" {
		return httpClient()
	}
	if cached, ok := authProxyClientCache.Load(proxyURL); ok {
		return cached.(*http.Client)
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: buildAuthTransport(proxyURL),
	}
	authProxyClientCache.Store(proxyURL, client)
	return client
}

func environmentProxy(req *http.Request) (*url.URL, error) {
	if req == nil || req.URL == nil {
		return nil, nil
	}
	proxy := os.Getenv(strings.ToUpper(req.URL.Scheme) + "_PROXY")
	if proxy == "" {
		proxy = os.Getenv(strings.ToLower(req.URL.Scheme) + "_proxy")
	}
	if proxy == "" {
		proxy = os.Getenv("ALL_PROXY")
		if proxy == "" {
			proxy = os.Getenv("all_proxy")
		}
	}
	if proxy == "" || proxyBypassed(req.URL.Hostname()) {
		return nil, nil
	}
	if !strings.Contains(proxy, "://") {
		proxy = "http://" + proxy
	}
	return url.Parse(proxy)
}

func proxyBypassed(host string) bool {
	noProxy := os.Getenv("NO_PROXY")
	if noProxy == "" {
		noProxy = os.Getenv("no_proxy")
	}
	host = strings.ToLower(strings.TrimSpace(host))
	for _, raw := range strings.Split(noProxy, ",") {
		pattern := strings.ToLower(strings.TrimSpace(raw))
		if pattern == "*" {
			return true
		}
		pattern = strings.TrimPrefix(strings.Split(pattern, ":")[0], ".")
		if pattern != "" && (host == pattern || strings.HasSuffix(host, "."+pattern)) {
			return true
		}
	}
	return false
}

// buildAuthTransport 构建带可选代理的 Transport
func buildAuthTransport(proxyURL string) *http.Transport {
	t := &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
	if proxyURL != "" {
		if u, err := url.Parse(proxyURL); err == nil {
			t.Proxy = http.ProxyURL(u)
			t.ForceAttemptHTTP2 = false
		}
	} else {
		t.Proxy = environmentProxy
	}
	return t
}

// InitHttpClient 初始化（或重新初始化）auth 模块的全局 HTTP 客户端
func InitHttpClient(proxyURL string) {
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: buildAuthTransport(proxyURL),
	}
	httpClientStore.Store(client)
}
