package main

import (
	"lili/handle"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	files := http.FileServer(http.Dir("./statics"))
	h := handle.New()
	mux.Handle("/statics/", http.StripPrefix("/statics/", files))
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/search", h.Search)
	mux.HandleFunc("/api/search", cos(h.SearchService, "application/json;charset=UTF-8", http.MethodGet))
	mux.HandleFunc("/api/stream", cos(h.Stream, "application/octet-stream;charset=UTF-8", http.MethodPost))
	mux.HandleFunc("/api/g", cos(h.Gapi, "application/json;charset=UTF-8", http.MethodPost))
	mux.HandleFunc("/api/gv", cos(h.Gv, "application/json;charset=UTF-8", http.MethodPost))
	mux.HandleFunc("/api/completions", cos(h.Completions, "application/json;charset=UTF-8", http.MethodPost))
	mux.HandleFunc("/api/generateImage", cos(h.GenerateImage, "application/json;charset=UTF-8", http.MethodPost))
	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}
	server.ListenAndServe()
}

func cos(handle func(http.ResponseWriter, *http.Request), contentType, method string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Access-Control-Allow-Origin", "*")                                // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
		rw.Header().Add("Access-Control-Allow-Credentials", "true")                        //设置为true，允许ajax异步请求带cookie信息
		rw.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE") //允许请求方法
		rw.Header().Set("Access-Control-Allow-Headers", "Content-Type,access-control-allow-origin, access-control-allow-headers,x-requested-with")
		// rw.Header().Set("content-type", "application/json;charset=UTF-8") //返回数据格式是json
		rw.Header().Set("Content-Type", contentType)
		// w.Header().Set("Content-Type", "text/event-stream")
		if req.Method == http.MethodOptions {
			rw.WriteHeader(http.StatusNoContent)
			return
		}
		if req.Method != method {
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handle(rw, req)
	}
}
