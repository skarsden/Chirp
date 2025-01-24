package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/skarsden/Chirp/internal/auth"
)

func (cfg *apiConfig) handlerUpdateUserChirpyRed(w http.ResponseWriter, r *http.Request) {
	type reqParams struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	apikey, err := auth.GetApiKey(r.Header)
	if err != nil {
		log.Printf("Couldn't get api key: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if apikey != cfg.polka_key {
		log.Printf("Invalid api key")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	req := reqParams{}
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&req)
	if err != nil {
		log.Printf("Couldn't decode json: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if req.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = cfg.queries.UpdateUserChirpyRed(r.Context(), req.Data.UserID)
	if err != nil {
		log.Printf("User not found: %s", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
