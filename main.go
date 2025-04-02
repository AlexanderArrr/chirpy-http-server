package main

import "net/http"

func main() {
	srvMux := http.NewServeMux()
	srvMux.Handle("/", http.FileServer(http.Dir(".")))
	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: srvMux,
	}

	srv.ListenAndServe()
}
