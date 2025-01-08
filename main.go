package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileServerHits atomic.Int32
}

func main() {
	const port = "8080"
	const root = "."

	//records number of handler calls
	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
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
