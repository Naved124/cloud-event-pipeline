package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

type Event struct {
	ID        int       `json:"id"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func initDB() {
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")

	if dbHost == "" {
		dbHost = "postgres"
	}
	if dbPort == "" {
		dbPort = "5432"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil && db.Ping() == nil {
			log.Println("Connected to PostgreSQL successfully.")
			break
		}
		log.Printf("Waiting for database connection (attempt %d/10)...", i+1)
		time.Sleep(2 * time.Second)
	}

	if db != nil {
		query := `CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			payload TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`
		db.Exec(query)
	}
}

func requireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("API_KEY")
		if apiKey == "" || r.Header.Get("X-API-Key") != apiKey {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dbStatus := "connected"
	if db == nil || db.Ping() != nil {
		dbStatus = "disconnected"
	}

	json.NewEncoder(w).Encode(map[string]string{
		"status":   "healthy",
		"database": dbStatus,
	})
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB cap

		var req struct {
			Payload string `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Payload == "" {
			http.Error(w, `{"error":"Invalid payload"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var id int
		err := db.QueryRowContext(ctx, "INSERT INTO events (payload) VALUES ($1) RETURNING id", req.Payload).Scan(&id)
		if err != nil {
			http.Error(w, `{"error":"Database write error"}`, http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Event recorded",
			"id":      id,
		})
		return
	}

	if r.Method == http.MethodGet {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rows, err := db.QueryContext(ctx, "SELECT id, payload, created_at FROM events ORDER BY id DESC LIMIT 10")
		if err != nil {
			http.Error(w, `{"error":"Database read error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var events []Event
		for rows.Next() {
			var e Event
			if err := rows.Scan(&e.ID, &e.Payload, &e.CreatedAt); err == nil {
				events = append(events, e)
			}
		}

		json.NewEncoder(w).Encode(events)
		return
	}

	http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
}

func main() {
	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/events", requireAPIKey(eventsHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		log.Printf("Server running on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	if db != nil {
		db.Close()
	}
}