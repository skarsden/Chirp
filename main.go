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
}

func main() {
	godotenv.Load()
	dbUrl := os.Getenv("DB_URL")
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
	}

	//Declare handler and register handler functions
	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(root)))))

	mux.HandleFunc("GET /api/healthz", handlerReady)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidate)

	//configure server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	//run server
	fmt.Printf("Serving on port :%s\n", port)
	server.ListenAndServe()
}
