package api

import (
	"fmt"
	"net/http"
)

func ApiProxy(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"login": "success"}`)
}
