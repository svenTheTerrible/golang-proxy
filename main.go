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
	"strings"
	"sync"
	"time"
)

type Lock struct {
	FailedTries   int
	LastFailedTry time.Time
}

var (
	lockList = map[string]Lock{}
	lockMu   sync.Mutex
)

// applyLockIfSecretIsWrong records a failed attempt for the given IP.
// The lockout until is LastFailedTry + (FailedTries * 1 minute).
func applyLockIfSecretIsWrong(ipAdress string) {
	lockMu.Lock()
	defer lockMu.Unlock()
	entry, exists := lockList[ipAdress]
	if exists {
		entry.FailedTries++
	} else {
		entry.FailedTries = 1
	}
	entry.LastFailedTry = time.Now()
	lockList[ipAdress] = entry
}

// checkIfIpIsLockedByLockList returns true if the IP is currently in a lockout period.
func checkIfIpIsLockedByLockList(ipAddress string) bool {
	lockMu.Lock()
	defer lockMu.Unlock()
	entry, exists := lockList[ipAddress]
	if !exists {
		return false
	}
	lockUntil := entry.LastFailedTry.Add(time.Duration(entry.FailedTries*2) * time.Minute)

	ipIsStillLocked := time.Now().Before(lockUntil)

	if ipIsStillLocked {
		log.Printf("IP address %s is locked until %s", ipAddress, lockUntil)
	}

	return ipIsStillLocked
}

// clearLock removes any lockout record for the IP (call after successful auth).
func clearLock(ipAddress string) {
	lockMu.Lock()
	defer lockMu.Unlock()
	delete(lockList, ipAddress)
}

// clientIP returns the most reliable client IP for the request.
// It prefers the leftmost entry of X-Forwarded-For (set by reverse proxies
// such as Traefik, nginx, or HAProxy) and falls back to the TCP peer address.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Leftmost entry is the original client per the common convention.
		for _, part := range strings.Split(xff, ",") {
			if parsed := net.ParseIP(strings.TrimSpace(part)); parsed != nil {
				return parsed.String()
			}
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

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

	requiredSecret := os.Getenv("PROXY_SECRET")

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Skip TLS verification for self-signed backend certificates.
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		if checkIfIpIsLockedByLockList(ip) {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "Forbidden: On cooldown for next try because of invalid or missing secret")
			return
		}

		providedSecret := r.Header.Get("X-API-Secret")
		secretWrong := requiredSecret != "" && providedSecret != requiredSecret
		if secretWrong {
			applyLockIfSecretIsWrong(ip)
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprintf(w, "Forbidden: invalid or missing secret")
			return
		}

		// Preserve the original Host header so backend can see where requests came from.
		if r.Host == "" || r.Host == targetURL.Host {
			if h := r.Header.Get("X-Forwarded-Host"); h != "" {
				r.Host = h
			}
		}

		// Add the real client IP if not already present (e.g. behind a
		// reverse proxy like Traefik, where RemoteAddr is the proxy's address).
		if r.Header.Get("X-Real-Ip") == "" {
			r.Header.Set("X-Real-Ip", ip)
		}

		// Ensure Origin is forwarded (already included by default, but explicit is safer).
		if origin := r.Header.Get("Origin"); origin != "" {
			r.Header.Set("Origin", origin)
		}

		clearLock(ip)

		proxy.ServeHTTP(w, r)
	})

	// Optional: log errors from upstream.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Backend unavailable")
	}

	addr := ":" + port
	log.Printf("starting proxy on %s -> %s (TLS verify disabled)", addr, targetURL.String())
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
