package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alexanderarrr/chirpy-http-server/internal/auth"
	"github.com/alexanderarrr/chirpy-http-server/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type returnVals struct {
		Id         uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email      string    `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode request: %v", err)
		return
	}

	_, err = cfg.dbQueries.GetUser(r.Context(), params.Email)
	if err == nil {
		respondWithError(w, http.StatusBadRequest, "Email already registered, can not create user", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "can't use password: %v", err)
		return
	}

	user, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error while creating user: %s", err)
		return
	}

	response := returnVals{
		Id:         user.ID,
		Created_at: user.CreatedAt,
		Updated_at: user.UpdatedAt,
		Email:      user.Email,
	}

	respondWithJSON(w, http.StatusCreated, response)
	return
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode request: %v", err)
		return
	}

	user, err := cfg.dbQueries.GetUser(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "incorrect email or password", err)
		return
	}

	err = auth.CheckPasswordHash(user.HashedPassword, params.Password)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "incorrect email or password", err)
		return
	}

	// Access Token
	expirationTime := time.Hour

	token, err := auth.MakeJWT(user.ID, cfg.tokenSecret, expirationTime)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while creating access token", err)
		return
	}

	// Refresh Token
	refreshTokenString, _ := auth.MakeRefreshToken()
	refreshTokenExpiration := time.Now().AddDate(0, 0, 60)

	refreshToken, err := cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshTokenString,
		UserID:    user.ID,
		ExpiresAt: refreshTokenExpiration,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while storing refresh token: %v", err)
		return
	}

	// JSON Response
	type returnVals struct {
		Id            uuid.UUID `json:"id"`
		Created_at    time.Time `json:"created_at"`
		Updated_at    time.Time `json:"updated_at"`
		Email         string    `json:"email"`
		Token         string    `json:"token"`
		Refresh_token string    `json:"refresh_token"`
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		Id:            user.ID,
		Created_at:    user.CreatedAt,
		Updated_at:    user.UpdatedAt,
		Email:         user.Email,
		Token:         token,
		Refresh_token: refreshToken.Token,
	})
}
func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Wrong / Invalid refresh token in header", err)
		return
	}

	refreshToken, err := cfg.dbQueries.GetRefreshToken(r.Context(), refreshTokenString)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Wrong / Invalid refresh token in header", err)
		return
	}

	user, err := cfg.dbQueries.GetUserFromRefreshToken(r.Context(), database.GetUserFromRefreshTokenParams{
		Token:     refreshToken.Token,
		ExpiresAt: time.Now(),
	})
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error while fetching user via refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, cfg.tokenSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error while creating access token", err)
	}

	type returnVals struct {
		Token string `json:"token"`
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		Token: accessToken,
	})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshTokenString, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Wrong / Invalid refresh token in header", err)
		return
	}

	_, err = cfg.dbQueries.RevokeRefreshToken(r.Context(), refreshTokenString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error while revoking refresh token", err)
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Malformed or missing access token", err)
	}

	userID, err := auth.ValidateJWT(accessToken, cfg.tokenSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Malformed or missing access token", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while decoding request", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while hashing password", err)
		return
	}

	user, err := cfg.dbQueries.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
		ID:             userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error while updating user", err)
		return
	}

	type returnVals struct {
		Id         uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email      string    `json:"email"`
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		Id:         user.ID,
		Created_at: user.CreatedAt,
		Updated_at: user.UpdatedAt,
		Email:      user.Email,
	})
}
