package main

import (
	"lili/handle"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("./statics"))
	mux.Handle("/statics/", http.StripPrefix("/statics/", files))
	mux.HandleFunc("/", handle.Index)
	mux.HandleFunc("/search", handle.Search)
	mux.HandleFunc("/api/stream", handle.Stream)

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}
	server.ListenAndServe()
}
