package store

import (
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// gormLoggerSilent retourne un logger GORM en mode silent
func gormLoggerSilent() gormLogger.Interface {
	return gormLogger.Default.LogMode(gormLogger.Info)
}

// Store contient le pool de connexions à la base de données.
type Store struct {
	gormDB *gorm.DB // Connexion GORM
}

// Gorm expose la connexion GORM pour des opérations directes si nécessaire.
func (s *Store) Gorm() *gorm.DB {
	return s.gormDB
}

// NewStore crée et retourne une nouvelle instance de notre Store avec un pool de connexions calibré.
func NewStore(connStr string) (*Store, error) {
	// Initialisation GORM avec logger en mode silent
	gormDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: gormLoggerSilent(),
	})
	if err != nil {
		return nil, err
	}

	// Configuration explicite du pool de connexions SQL sous-jacent
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", 100)
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 20)
	connMaxLifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute)

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	return &Store{gormDB: gormDB}, nil
}

func getEnvInt(key string, defaultVal int) int {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil && val > 0 {
			return val
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if valStr := os.Getenv(key); valStr != "" {
		if d, err := time.ParseDuration(valStr); err == nil && d > 0 {
			return d
		}
	}
	return defaultVal
}

// Close ferme le pool de connexions.
func (s *Store) Close() {
	sqlDB, err := s.gormDB.DB()
	if err == nil {
		sqlDB.Close()
	}
}
