package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"notes-app/internal/models"
	"notes-app/pkg/utils"

	"github.com/gorilla/mux"
)

func CreateNoteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)

		var note struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if len(note.Title) == 0 || len(note.Content) == 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "Title and content are required")
			return
		}

		var noteID int
		err := db.QueryRow(
			"INSERT INTO notes (title, content, user_id) VALUES ($1, $2, $3) RETURNING id",
			note.Title, note.Content, userID,
		).Scan(&noteID)

		if err != nil {
			log.Printf("Database error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create note")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, map[string]int{"id": noteID})
	}
}

func GetNotesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)

		rows, err := db.Query(`
			SELECT id, title, content, created_at, updated_at 
			FROM notes 
			WHERE user_id = $1
			ORDER BY updated_at DESC
		`, userID)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch notes")
			return
		}
		defer rows.Close()

		var notes []models.Note
		for rows.Next() {
			var note models.Note
			if err := rows.Scan(
				&note.ID,
				&note.Title,
				&note.Content,
				&note.CreatedAt,
				&note.UpdatedAt,
			); err != nil {
				utils.RespondWithError(w, http.StatusInternalServerError, "Failed to read note data")
				return
			}
			note.UserID = userID
			notes = append(notes, note)
		}

		utils.RespondWithJSON(w, http.StatusOK, notes)
	}
}

func GetNoteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)
		vars := mux.Vars(r)
		noteID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid note ID")
			return
		}

		var note models.Note
		err = db.QueryRow(`
			SELECT id, title, content, created_at, updated_at 
			FROM notes 
			WHERE id = $1 AND user_id = $2
		`, noteID, userID).Scan(
			&note.ID,
			&note.Title,
			&note.Content,
			&note.CreatedAt,
			&note.UpdatedAt,
		)

		if err != nil {
			if err == sql.ErrNoRows {
				utils.RespondWithError(w, http.StatusNotFound, "Note not found")
			} else {
				utils.RespondWithError(w, http.StatusInternalServerError, "Failed to get note")
			}
			return
		}
		note.UserID = userID

		utils.RespondWithJSON(w, http.StatusOK, note)
	}
}

func UpdateNoteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)
		vars := mux.Vars(r)
		noteID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid note ID")
			return
		}

		var note struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request format")
			return
		}

		if strings.TrimSpace(note.Title) == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Title cannot be empty")
			return
		}

		result, err := db.Exec(`
			UPDATE notes 
			SET title = $1, content = $2, updated_at = NOW() 
			WHERE id = $3 AND user_id = $4
		`, note.Title, note.Content, noteID, userID)

		if err != nil {
			log.Printf("Database error: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update note")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "Note not found or not owned by user")
			return
		}

		var updatedNote models.Note
		err = db.QueryRow(`
			SELECT id, title, content, created_at, updated_at 
			FROM notes 
			WHERE id = $1
		`, noteID).Scan(
			&updatedNote.ID,
			&updatedNote.Title,
			&updatedNote.Content,
			&updatedNote.CreatedAt,
			&updatedNote.UpdatedAt,
		)

		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch updated note")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, updatedNote)
	}
}

func DeleteNoteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value("user_id").(int)
		vars := mux.Vars(r)
		noteID, err := strconv.Atoi(vars["id"])
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid note ID")
			return
		}

		result, err := db.Exec(`
			DELETE FROM notes 
			WHERE id = $1 AND user_id = $2
		`, noteID, userID)

		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete note")
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			utils.RespondWithError(w, http.StatusNotFound, "Note not found")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Note deleted successfully"})
	}
}
