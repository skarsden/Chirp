package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skarsden/Chirp/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

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
	if cfg.platform != "dev" {
		log.Printf("Not running on development machine")
		w.WriteHeader(http.StatusForbidden)
		return
	}
	cfg.queries.DeleteUsers(r.Context())
	cfg.fileServerHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func (cfg *apiConfig) handlerPostChirp(w http.ResponseWriter, r *http.Request) {
	type chirpParams struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
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

	chirpResp, err := cfg.queries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   strings.Join(stringSlice, " "),
		UserID: chirp.UserID,
	})
	if err != nil {
		log.Printf("Error posting creating chirp: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//text is valid, set response to valid and marshal it, and write it back
	respBody := Chirp{
		ID:        chirpResp.ID,
		CreatedAt: chirpResp.CreatedAt,
		UpdatedAt: chirpResp.UpdatedAt,
		Body:      chirpResp.Body,
		UserID:    chirp.UserID,
	}

	data, err := json.Marshal(respBody)

	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.queries.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error getting chirps: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.ID,
		})
	}

	data, err := json.Marshal(chirps)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)

}

func (cfg *apiConfig) handlerGetChirpById(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		log.Printf("Chirp ID not found\n")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dbChirp, err := cfg.queries.GetChirp(r.Context(), chirpID)

	chirp := Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	}

	data, err := json.Marshal(chirp)
	if err != nil {
		log.Printf("Error marhsalling json: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Conten-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type reqParams struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	req := reqParams{}
	err := decoder.Decode(&req)
	if err != nil {
		log.Printf("Error decoding json request: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.queries.CreateUser(r.Context(), req.Email)
	if err != nil {
		log.Printf("Error creating user: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}
