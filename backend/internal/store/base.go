package store

import (
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

// NewStore crée et retourne une nouvelle instance de notre Store.
func NewStore(connStr string) (*Store, error) {
	// Initialisation GORM avec logger en mode silent
	gormDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: gormLoggerSilent(),
	})
	if err != nil {
		return nil, err
	}
	return &Store{gormDB: gormDB}, nil
}

// Close ferme le pool de connexions.
func (s *Store) Close() {
	sqlDB, err := s.gormDB.DB()
	if err == nil {
		sqlDB.Close()
	}
}
