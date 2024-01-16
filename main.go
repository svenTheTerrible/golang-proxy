package main

import (
	"fmt"
	"go-proxy/api"
	"net/http"
	"regexp"
)

func main() {
	handler := &api.RegexpHandler{}
	staticFs := http.FileServer(http.Dir("./static/"))

	apiPath, _ := regexp.Compile("^/api.*")
	handler.HandleFunc(apiPath, api.ApiProxy)

	rootPath, _ := regexp.Compile("/")
	handler.Handler(rootPath, staticFs)

	fmt.Println("Server running on port 3000")
	http.ListenAndServe(":3000", handler)
}
