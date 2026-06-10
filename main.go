package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	targetEnv := os.Getenv("PROXY_TARGET_URL")
	if targetEnv == "" {
		log.Fatal("PROXY_TARGET_URL is required (e.g. http://10.0.0.5:8080)")
	}

	targetURL, err := url.Parse(targetEnv)
	if err != nil {
		log.Fatalf("invalid PROXY_TARGET_URL: %v", err)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "3000"
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Skip TLS verification for self-signed backend certificates.
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		},
	}

	// Customize director to preserve important request metadata.
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)

		// Preserve the original Host header so backend can see where requests came from.
		if r.Host == "" || r.Host == targetURL.Host {
			if h := r.Header.Get("X-Forwarded-Host"); h != "" {
				r.Host = h
			}
		}

		// Add standard forwarding headers if not already present.
		// These are commonly used in Docker/K8s and reverse-proxy setups.
		if r.Header.Get("X-Real-Ip") == "" {
			if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && ip != "" {
				r.Header.Set("X-Real-Ip", ip)
			}
		}

		// Ensure Origin is forwarded (already included by default, but explicit is safer).
		if origin := r.Header.Get("Origin"); origin != "" {
			r.Header.Set("Origin", origin)
		}
	}

	// Optional: log errors from upstream.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Backend unavailable")
	}

	addr := ":" + port
	log.Printf("starting proxy on %s -> %s (TLS verify disabled)", addr, targetURL.String())
	if err := http.ListenAndServe(addr, proxy); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
