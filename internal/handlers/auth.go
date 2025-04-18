package handlers

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"notes-app/internal/models"
	"notes-app/pkg/utils"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(db *sql.DB, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.Printf("Registration request body: %s", string(body))

		var request struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.Unmarshal(body, &request); err != nil {
			log.Printf("JSON decode error: %v", err)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request format")
			return
		}

		username := strings.TrimSpace(request.Username)
		password := strings.TrimSpace(request.Password)

		if len(username) < 3 || len(password) < 6 {
			utils.RespondWithError(w, http.StatusBadRequest, "Username must be at least 3 chars and password 6 chars")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Password hash error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Registration failed")
			return
		}

		var userID int
		err = db.QueryRow(
			"INSERT INTO users (username, password) VALUES ($1, $2) RETURNING id",
			username, string(hashedPassword),
		).Scan(&userID)

		if err != nil {
			log.Printf("Database error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Registration failed")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"id":       userID,
			"username": username,
		})
	}
}

func LoginHandler(db *sql.DB, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		var user models.User
		err := db.QueryRow(
			"SELECT id, username, password FROM users WHERE username = $1",
			creds.Username,
		).Scan(&user.ID, &user.Username, &user.Password)

		if err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, models.Claims{
			UserID: user.ID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			},
		})

		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"token":    tokenString,
			"user_id":  user.ID,
			"username": user.Username,
		})
	}
}
