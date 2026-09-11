package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	db             *sql.DB
	allowedOrigins map[string]struct{}
}

type response map[string]any

func main() {
	port := env("PORT", "8080")
	db, err := openDatabase(os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Fatalf("database configuration error: %v", err)
	}
	if db != nil {
		defer db.Close()
	}

	app := &application{
		db:             db,
		allowedOrigins: parseOrigins(env("CORS_ALLOWED_ORIGINS", "http://localhost:5173,https://zhoujie0420.github.io")),
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("club api listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", app.health)
	mux.HandleFunc("GET /readyz", app.ready)
	mux.HandleFunc("GET /api/v1/meta", app.meta)
	return app.cors(app.logRequest(mux))
}

func (app *application) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{"status": "ok", "service": "club-api"})
}

func (app *application) ready(w http.ResponseWriter, r *http.Request) {
	if app.db == nil {
		writeJSON(w, http.StatusServiceUnavailable, response{"status": "not_ready", "database": "not_configured"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := app.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, response{"status": "not_ready", "database": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "ready", "database": "ok"})
}

func (app *application) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{
		"name":    "HOLE CLUB API",
		"version": env("APP_VERSION", "dev"),
	})
}

func (app *application) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := app.allowedOrigins[origin]; origin != "" && ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}

func openDatabase(dsn string) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

func parseOrigins(value string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, origin := range strings.Split(value, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			result[origin] = struct{}{}
		}
	}
	return result
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("encode response: %v", err)
	}
}
