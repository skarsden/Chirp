package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skarsden/Chirp/internal/auth"
	"github.com/skarsden/Chirp/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

// Ready
func handlerReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

// Middleware for metrics
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// Metrics
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

// Reset
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
			ID:        dbUser.ID,
			Email:     dbUser.Email,
			CreatedAt: dbUser.CreatedAt,
			UpdatedAt: dbUser.UpdatedAt,
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

// Update User
func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type reqParams struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type Response struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
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

	user, err := cfg.queries.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          req.Email,
		HashedPassword: hashedPass,
		ID:             userID,
	})

	resp := Response{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: time.Now().UTC(),
		Email:     user.Email,
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
