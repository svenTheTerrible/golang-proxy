package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func ApiProxy(w http.ResponseWriter, r *http.Request) {
	url, _ := url.Parse("http://192.168.178.56/heimkino/api/")
	proxy := httputil.NewSingleHostReverseProxy(url)
	r.URL.Path = strings.Replace(r.URL.Path, "/api", "", 1)
	r.RequestURI = strings.Replace(r.RequestURI, "/api", "", 1)
	proxy.ServeHTTP(w, r)
}
