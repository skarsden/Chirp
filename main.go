package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/skarsden/Chirp/internal/database"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	queries        *database.Queries
	platform       string
	secret         string
	polka_key      string
}

func main() {
	//loading env variables
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	secret := os.Getenv("SECRET")
	polka_key := os.Getenv("POLKA_KEY")

	//open db connection
	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Printf("Error opening sql database: %s", err)
	}
	dbQueries := database.New(db)

	const port = "8080"
	const root = "."

	//records number of handler calls
	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
		queries:        dbQueries,
		platform:       platform,
		secret:         secret,
		polka_key:      polka_key,
	}

	//Declare handler and register handler functions
	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(root)))))

	//meta endpoints
	mux.HandleFunc("GET /api/ready", handlerReady)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	//chirp endpoints
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerPostChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpById)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirpById)

	//user endpoints
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUserPassword)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	//token endpoints
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefreshToken)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevokeToken)

	//webhook endpoint
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerUpdateUserChirpyRed)

	//configure server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	//run server
	fmt.Printf("Serving on port :%s\n", port)
	server.ListenAndServe()
}
