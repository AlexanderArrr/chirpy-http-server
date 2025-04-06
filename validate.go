package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type returnVals struct {
		Cleaned_body string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	cleanedBody := cleanChirp(params.Body)

	respondWithJSON(w, http.StatusOK, returnVals{
		Cleaned_body: cleanedBody,
	})

}

func cleanChirp(body string) string {
	profanity := []string{"kerfuffle", "sharbert", "fornax"}
	splitString := strings.Split(body, " ")
	for i, ch := range splitString {
		for _, prof := range profanity {
			if strings.ToLower(ch) == prof {
				splitString[i] = "****"
			}
		}
	}
	return strings.Join(splitString, " ")
}
