package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"travel-ar-backend/internal/model"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	GetUserByEmail(email string) (*model.User, error)
	CreateUser(user model.User) (*model.User, error)
	UpdateUserGoogleInfo(userID int, googleID, avatar string) error
	GetUserByID(userID int) (*model.User, error)
	SaveRefreshToken(userID int, refreshToken string, expiresAt time.Time) error
	GetRefreshToken(token string) (*model.RefreshToken, error)
}

type service struct {
	db *sql.DB
}

var (
	database   = os.Getenv("BLUEPRINT_DB_DATABASE")
	password   = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username   = os.Getenv("BLUEPRINT_DB_USERNAME")
	port       = os.Getenv("BLUEPRINT_DB_PORT")
	host       = os.Getenv("BLUEPRINT_DB_HOST")
	schema     = os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", database)
	return s.db.Close()
}

func (s *service) GetUserByEmail(email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT user_id, email, google_id, name, avatar, provider, status FROM users WHERE email = $1`
	err := s.db.QueryRow(query, email).Scan(
		&user.UserID, &user.Email, &user.GoogleID, &user.Name, &user.Avatar, &user.Provider, &user.Status,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) CreateUser(user model.User) (*model.User, error) {
	query := `
        INSERT INTO users (email, google_id, name, avatar, provider, status, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())
        RETURNING user_id
    `
	err := s.db.QueryRow(query, user.Email, user.GoogleID, user.Name, user.Avatar, user.Provider, user.Status).
		Scan(&user.UserID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *service) UpdateUserGoogleInfo(userID int, googleID, avatar string) error {
	query := `UPDATE users SET google_id = $1, avatar = $2, updated_at = NOW() WHERE user_id = $3`
	_, err := s.db.Exec(query, googleID, avatar, userID)
	return err
}

func (s *service) GetUserByID(userID int) (*model.User, error) {
	user := &model.User{}
	query := `SELECT user_id, email, google_id, name, avatar, provider, status FROM users WHERE user_id = $1`
	err := s.db.QueryRow(query, userID).Scan(
		&user.UserID, &user.Email, &user.GoogleID, &user.Name, &user.Avatar, &user.Provider, &user.Status,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) SaveRefreshToken(userID int, refreshToken string, expiresAt time.Time) error {
	query := `
        INSERT INTO refresh_tokens (user_id, refresh_token, expires_at)
        VALUES ($1, $2, $3)
    `
	_, err := s.db.Exec(query, userID, refreshToken, expiresAt)
	return err
}

func (s *service) GetRefreshToken(token string) (*model.RefreshToken, error) {
	var rt model.RefreshToken
	query := `
        SELECT token_id, user_id, refresh_token, expires_at, created_at, revoked
        FROM refresh_tokens
        WHERE refresh_token = $1 AND revoked = FALSE
    `
	err := s.db.QueryRow(query, token).Scan(
		&rt.TokenID, &rt.UserID, &rt.RefreshToken, &rt.ExpiresAt, &rt.CreatedAt, &rt.Revoked,
	)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}
