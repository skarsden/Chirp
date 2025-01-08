package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func handlerReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`
		<html>

			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>

		</html>
	`, cfg.fileServerHits.Load())))
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileServerHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func handlerValidate(w http.ResponseWriter, r *http.Request) {
	type chirpParams struct {
		Body string `json:"body"`
	}

	type returnVals struct {
		CleanedBody string `json:"cleaned_body"`
	}

	//decode request JSON into struct
	decoder := json.NewDecoder(r.Body)
	chirp := chirpParams{}
	err := decoder.Decode(&chirp)
	if err != nil {
		log.Printf("Error decoding json: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//verify that text does not exceed 140 characters
	if len(chirp.Body) > 140 {
		log.Printf("Chirp exceeds 140 characters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	stringSlice := strings.Split(chirp.Body, " ")
	for i, s := range stringSlice {
		if strings.ToLower(s) == "kerfuffle" || strings.ToLower(s) == "sharbert" || strings.ToLower(s) == "fornax" {
			stringSlice[i] = "****"
		}
	}

	//text is valid, set response to valid and marshal it, and write it back
	respBody := returnVals{
		CleanedBody: strings.Join(stringSlice, " "),
	}

	data, err := json.Marshal(respBody)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
