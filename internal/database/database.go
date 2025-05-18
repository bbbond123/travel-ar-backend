package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"travel-ar-backend/internal/model"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	db *gorm.DB
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
	if dbInstance != nil {
		return dbInstance
	}
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s", host, username, password, database, port, schema)
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
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
	db, err := s.db.DB()
	stats := make(map[string]string)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err)
		return stats
	}
	err = db.Ping()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err)
		return stats
	}
	stats["status"] = "up"
	stats["message"] = "It's healthy"
	dbStats := db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)
	if dbStats.OpenConnections > 40 {
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
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	log.Printf("Disconnected from database: %s", database)
	return db.Close()
}

func (s *service) GetUserByEmail(email string) (*model.User, error) {
	user := &model.User{}
	err := s.db.Where("email = ?", email).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) CreateUser(user model.User) (*model.User, error) {
	err := s.db.Create(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *service) UpdateUserGoogleInfo(userID int, googleID, avatar string) error {
	return s.db.Model(&model.User{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"google_id":  googleID,
		"avatar":     avatar,
		"updated_at": time.Now(),
	}).Error
}

func (s *service) GetUserByID(userID int) (*model.User, error) {
	user := &model.User{}
	err := s.db.Where("user_id = ?", userID).First(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) SaveRefreshToken(userID int, refreshToken string, expiresAt time.Time) error {
	rt := model.RefreshToken{
		UserID:       userID,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		Revoked:      false,
	}
	return s.db.Create(&rt).Error
}

func (s *service) GetRefreshToken(token string) (*model.RefreshToken, error) {
	rt := &model.RefreshToken{}
	err := s.db.Where("refresh_token = ? AND revoked = FALSE", token).First(rt).Error
	if err != nil {
		return nil, err
	}
	return rt, nil
}
