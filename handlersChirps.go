package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skarsden/Chirp/internal/auth"
	"github.com/skarsden/Chirp/internal/database"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

// Post Chirp
func (cfg *apiConfig) handlerPostChirp(w http.ResponseWriter, r *http.Request) {
	type chirpParams struct {
		Body string `json:"body"`
	}

	//Get access token and validate it with user
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find access token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		log.Printf("Couldn't validate access token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//decode request JSON into struct
	decoder := json.NewDecoder(r.Body)
	chirp := chirpParams{}
	err = decoder.Decode(&chirp)
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

	//look for "profanity" and clean body
	stringSlice := strings.Split(chirp.Body, " ")
	for i, s := range stringSlice {
		if strings.ToLower(s) == "kerfuffle" || strings.ToLower(s) == "sharbert" || strings.ToLower(s) == "fornax" {
			stringSlice[i] = "****"
		}
	}

	//post body to database
	chirpResp, err := cfg.queries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   strings.Join(stringSlice, " "),
		UserID: userID,
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
		UserID:    userID,
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

// Get Chirps
func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.queries.GetChirps(r.Context())
	if err != nil {
		log.Printf("Error getting chirps: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	authorID := uuid.Nil
	authorIDString := r.URL.Query().Get("author_id")
	if authorIDString != "" {
		authorID, err = uuid.Parse(authorIDString)
		if err != nil {
			log.Printf("Invalid author ID: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	chirps := []Chirp{}
	for _, dbChirp := range dbChirps {
		log.Println(authorIDString)
		if authorID != uuid.Nil && dbChirp.UserID != authorID {
			continue
		}
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		})
	}

	sortString := r.URL.Query().Get("sort")
	if sortString == "asc" || sortString == "" {
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.Before(chirps[j].CreatedAt) })
	}
	if sortString == "desc" {
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.After(chirps[j].CreatedAt) })
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

// Get Chirp by ID
func (cfg *apiConfig) handlerGetChirpById(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		log.Printf("Invalid chirpd ID\n")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dbChirp, err := cfg.queries.GetChirp(r.Context(), chirpID)
	if err != nil {
		log.Printf("Couldn't get chirp: %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Delete Chirp
func (cfg *apiConfig) handlerDeleteChirpById(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)
	if err != nil {
		log.Printf("Invalid chirp ID: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find access token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		log.Printf("Could't validate access token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	dbChirp, err := cfg.queries.GetChirp(r.Context(), chirpID)
	if err != nil {
		log.Printf("Couldn't find chirp: %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if dbChirp.UserID != userID {
		log.Printf("You cannot delete this chirp: %s\n", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = cfg.queries.DeleteChirpById(r.Context(), chirpID)
	if err != nil {
		log.Printf("Couldn't delete chirp: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
