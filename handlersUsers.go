package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/skarsden/Chirp/internal/auth"
	"github.com/skarsden/Chirp/internal/database"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	Token       string    `json:"token"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

// Create User
func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type reqParams struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	req := reqParams{}
	err := decoder.Decode(&req)
	if err != nil {
		log.Printf("Error decoding json request: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashed_password, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.queries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          req.Email,
		HashedPassword: hashed_password,
	})
	if err != nil {
		log.Printf("Error creating user: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := User{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
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

// Login
func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type reqParams struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type Response struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	req := reqParams{}
	err := decoder.Decode(&req)
	if err != nil {
		log.Printf("Error decoding json: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//get user from database and verify password
	dbUser, err := cfg.queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Print("Incorrect email or password\n", req.Email)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = auth.CheckPassword(req.Password, dbUser.HashedPassword)
	if err != nil {
		log.Print("Incorrect email or password\n")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//make JWT
	accessToken, err := auth.MakeJWT(dbUser.ID, cfg.secret, time.Hour)
	if err != nil {
		log.Printf("Couldn't create access token: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//make refresh token
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Couldn't make refresh token: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = cfg.queries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		UserID:    dbUser.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 24 * 60),
	})
	if err != nil {
		log.Printf("Couldn't save refresh token: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := Response{
		User: User{
			ID:          dbUser.ID,
			Email:       dbUser.Email,
			CreatedAt:   dbUser.CreatedAt,
			UpdatedAt:   dbUser.UpdatedAt,
			IsChirpyRed: dbUser.IsChirpyRed,
		},
		Token:        accessToken,
		RefreshToken: refreshToken,
	}

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

// Update User Password
func (cfg *apiConfig) handlerUpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	type reqParams struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type Response struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

	decoder := json.NewDecoder(r.Body)
	req := reqParams{}
	err := decoder.Decode(&req)
	if err != nil {
		log.Printf("Error decoding json: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		log.Printf("Couldn't find access token: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		log.Printf("Couldn't validate access token: %s\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hashedPass, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Couldn't hash new password: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.queries.UpdateUserPassword(r.Context(), database.UpdateUserPasswordParams{
		Email:          req.Email,
		HashedPassword: hashedPass,
		ID:             userID,
	})

	resp := Response{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   time.Now().UTC(),
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
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
