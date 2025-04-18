package app

import (
	"database/sql"
	"log"
	"net/http"

	"notes-app/internal/config"
	"notes-app/internal/handlers"
	"notes-app/internal/middleware"
	"notes-app/internal/storage"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	config *config.Config
	db     *sql.DB
	router *mux.Router
}

func New(cfg *config.Config) (*App, error) {
	db, err := storage.InitDB(cfg)
	if err != nil {
		return nil, err
	}

	app := &App{
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
	}

	app.registerHandlers()

	return app, nil
}

func (a *App) registerHandlers() {
	// API routes
	apiRouter := a.router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/register", handlers.RegisterHandler(a.db, a.config.JWTSecret)).Methods("POST")
	apiRouter.HandleFunc("/login", handlers.LoginHandler(a.db, a.config.JWTSecret)).Methods("POST")

	// Authenticated API routes
	authRouter := apiRouter.PathPrefix("").Subrouter()
	authRouter.Use(middleware.AuthMiddleware(a.config.JWTSecret))

	authRouter.HandleFunc("/notes", handlers.CreateNoteHandler(a.db)).Methods("POST")
	authRouter.HandleFunc("/notes", handlers.GetNotesHandler(a.db)).Methods("GET")
	authRouter.HandleFunc("/notes/{id:[0-9]+}", handlers.GetNoteHandler(a.db)).Methods("GET")
	authRouter.HandleFunc("/notes/{id:[0-9]+}", handlers.UpdateNoteHandler(a.db)).Methods("PUT")
	authRouter.HandleFunc("/notes/{id:[0-9]+}", handlers.DeleteNoteHandler(a.db)).Methods("DELETE")

	// Web interface
	a.router.HandleFunc("/", handlers.IndexHandler)
	a.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Middleware
	// a.router.Use(middleware.CorsMiddleware)
}

func (a *App) Run() error {
	log.Printf("Server starting on http://localhost:%s", a.config.Port)
	return http.ListenAndServe(":"+a.config.Port, a.router)
}

func (a *App) Close() {
	a.db.Close()
}
