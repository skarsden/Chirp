package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/skarsden/Chirp/internal/auth"
)

// Refresh Token
func (cfg *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	type Response struct {
		Token string `json:"token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find refresh token: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := cfg.queries.GetUserFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("Couldn't get user from refresh token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)
	if err != nil {
		log.Printf("Coudln't validate token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	resp := Response{Token: accessToken}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshalling json: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (cfg *apiConfig) handlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find token: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	_, err = cfg.queries.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Printf("Error revoking session: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return

	}

	w.WriteHeader(http.StatusNoContent)
}
