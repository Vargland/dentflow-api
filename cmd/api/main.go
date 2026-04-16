// Package main is the entry point for the DentFlow Go API server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"

	db "github.com/psi-germanr/dentflow-api/internal/db/sqlc"
	"github.com/psi-germanr/dentflow-api/internal/middleware"
	"github.com/psi-germanr/dentflow-api/internal/modules/appointments"
	"github.com/psi-germanr/dentflow-api/internal/modules/auth"
	"github.com/psi-germanr/dentflow-api/internal/modules/evolutions"
	"github.com/psi-germanr/dentflow-api/internal/modules/patients"
	"github.com/psi-germanr/dentflow-api/internal/modules/settings"
)

func main() {
	// Load .env (ignore error in production — env vars set externally)
	_ = godotenv.Load()

	port := envOrDefault("PORT", "8080")
	dbURL := mustEnv("DATABASE_URL")
	authSecret := mustEnv("AUTH_SECRET")
	allowedOrigins := envOrDefault("ALLOWED_ORIGINS", "http://localhost:3000")

	// Database pool
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")

	// Wire up modules
	queries := db.New(pool)

	authRepo := auth.NewRepository(queries)
	authHandler := auth.NewHandler(authRepo)

	patientRepo := patients.NewRepository(pool)
	patientHandler := patients.NewHandler(patientRepo)

	evolutionRepo := evolutions.NewRepository(pool)
	evolutionHandler := evolutions.NewHandler(evolutionRepo)

	settingsRepo := settings.NewRepository(queries)
	settingsHandler := settings.NewHandler(settingsRepo)

	appointmentRepo := appointments.NewRepository(queries)
	appointmentHandler := appointments.NewHandler(appointmentRepo, queries)

	// Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.Recoverer)
	r.Use(middleware.Logger)

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   splitComma(allowedOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})
	r.Use(c.Handler)

	// Health check (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth endpoints (no JWT required — these issue tokens)
	authHandler.RegisterRoutes(r)

	// API v1 — all routes require JWT auth
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.Auth(authSecret))

		// Settings (timezone + Google Calendar)
		settingsHandler.RegisterProtectedRoutes(r)

		// Appointments
		r.Route("/appointments", func(r chi.Router) {
			appointmentHandler.RegisterRoutes(r)
		})

		// Patients
		r.Route("/patients", func(r chi.Router) {
			patientHandler.RegisterRoutes(r)

			// Evolutions nested under patients
			r.Route("/{id}/evolutions", func(r chi.Router) {
				evolutionHandler.RegisterRoutes(r)
			})
		})
	})

	// Google Calendar OAuth — public routes (browser redirects, no bearer token)
	r.Get("/auth/google/calendar", settingsHandler.StartCalendarOAuth)
	r.Get("/auth/google/calendar/callback", settingsHandler.CalendarOAuthCallback)

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start in background
	go func() {
		log.Printf("DentFlow API listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("stopped")
}

// mustEnv returns the value of an env var or fatals.
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

// envOrDefault returns the env var value or a fallback.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitComma splits a comma-separated string into a slice.
func splitComma(s string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}
