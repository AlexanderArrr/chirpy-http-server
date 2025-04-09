package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/alexanderarrr/chirpy-http-server/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	const filepathRoot = "."
	const port = "8080"

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error while accessing database: %v", err)
		return
	}
	platform := os.Getenv("PLATFORM")
	tokenSecret := os.Getenv("TOKEN_SECRET")

	apiCfg := &apiConfig{
		dbQueries:   *database.New(db),
		platform:    platform,
		tokenSecret: tokenSecret,
	}

	srvMux := http.NewServeMux()
	srvMux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))))
	srvMux.HandleFunc("GET /api/healthz", handlerReadiness)
	srvMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	srvMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	srvMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	srvMux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)
	srvMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	srvMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	srvMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
	srvMux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirp)
	srvMux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	srvMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)

	srv := &http.Server{
		Addr:    "localhost:" + port,
		Handler: srvMux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
