package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	db             *sql.DB
	allowedOrigins map[string]struct{}
	store          *clubStore
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
	app.store, err = newClubStore(env("CLUB_DB_PATH", "club.db"), os.Getenv("CLUB_ACCESS_CODE"))
	if err != nil {
		log.Fatal(err)
	}
	defer app.store.db.Close()

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
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery(), app.logRequest(), app.cors())
	engine.GET("/healthz", app.health)
	engine.GET("/readyz", app.ready)
	engine.GET("/api/v1/meta", app.meta)
	if app.store != nil {
		app.registerClub(engine)
		webDir := env("CLUB_WEB_DIR", "web")
		engine.GET("/club", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/club/")
		})
		engine.GET("/club/*filepath", gin.WrapH(http.StripPrefix("/club/", http.FileServer(http.Dir(webDir)))))
		engine.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, "/club/")
		})
	}
	return engine
}

func (app *application) health(c *gin.Context) {
	writeJSON(c, http.StatusOK, response{"status": "ok", "service": "club-api"})
}

func (app *application) ready(c *gin.Context) {
	if app.store != nil {
		if err := app.store.db.PingContext(c.Request.Context()); err == nil {
			writeJSON(c, http.StatusOK, response{"status": "ready", "database": "sqlite"})
			return
		}
	}
	if app.db == nil {
		writeJSON(c, http.StatusServiceUnavailable, response{"status": "not_ready", "database": "not_configured"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := app.db.PingContext(ctx); err != nil {
		writeJSON(c, http.StatusServiceUnavailable, response{"status": "not_ready", "database": "unavailable"})
		return
	}
	writeJSON(c, http.StatusOK, response{"status": "ready", "database": "ok"})
}

func (app *application) meta(c *gin.Context) {
	writeJSON(c, http.StatusOK, response{
		"name":    "HOLE CLUB API",
		"version": env("APP_VERSION", "dev"),
	})
}

func (app *application) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := app.allowedOrigins[origin]; origin != "" && ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (app *application) logRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		log.Printf("%s %s %s", c.Request.Method, c.Request.URL.Path, time.Since(started).Round(time.Millisecond))
	}
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

func writeJSON(c *gin.Context, status int, body response) {
	c.JSON(status, body)
}
