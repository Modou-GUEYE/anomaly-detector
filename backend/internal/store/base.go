package store

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// getGormLogger configure le niveau de log GORM. Par défaut: Silent pour des performances maximales.
func getGormLogger() gormLogger.Interface {
	logLevel := os.Getenv("DB_LOG_LEVEL")
	switch strings.ToLower(logLevel) {
	case "info":
		return gormLogger.Default.LogMode(gormLogger.Info)
	case "warn":
		return gormLogger.Default.LogMode(gormLogger.Warn)
	case "error":
		return gormLogger.Default.LogMode(gormLogger.Error)
	default:
		return gormLogger.Default.LogMode(gormLogger.Silent)
	}
}

// Store contient le pool de connexions à la base de données et les caches en mémoire.
type Store struct {
	gormDB       *gorm.DB
	refCache     *RefCache
	accountCache sync.Map // cache en mémoire: numeroCompte -> *models.Compte
	accountCount atomic.Int64
	accountMax   int64

	userCache    sync.Map // cache en mémoire: id utilisateur -> userCacheEntry
	userCacheTTL time.Duration

	metricsCache sync.Map // cache court des agrégats du tableau de bord
}

// Gorm expose la connexion GORM pour des opérations directes si nécessaire.
func (s *Store) Gorm() *gorm.DB {
	return s.gormDB
}

// NewStore crée et retourne une nouvelle instance de notre Store avec un pool de connexions calibré.
func NewStore(connStr string) (*Store, error) {
	// Initialisation GORM avec logger silencieux par défaut (configurable via DB_LOG_LEVEL)
	gormDB, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: getGormLogger(),
	})
	if err != nil {
		return nil, err
	}

	// Configuration explicite du pool de connexions SQL sous-jacent
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}

	// Le pool doit rester nettement sous le max_connections de PostgreSQL (100 par défaut)
	// pour éviter l'erreur SQLSTATE 53300 « too many clients ». Les connexions inactives
	// sont conservées (idle = open) : en fermer sous charge force PostgreSQL à recréer un
	// processus serveur à chaque pic, ce qui coûte plusieurs millisecondes par requête.
	maxOpenConns := clampToServerLimit(gormDB, getEnvInt("DB_MAX_OPEN_CONNS", 50))
	maxIdleConns := min(getEnvInt("DB_MAX_IDLE_CONNS", maxOpenConns), maxOpenConns)
	connMaxLifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	connMaxIdleTime := getEnvDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	store := &Store{
		gormDB:       gormDB,
		refCache:     newRefCache(),
		accountMax:   int64(getEnvInt("ACCOUNT_CACHE_MAX", 500000)),
		userCacheTTL: getEnvDuration("USER_CACHE_TTL", 30*time.Second),
	}

	return store, nil
}

// dbConnSafetyMargin connexions laissées libres pour l'administration, la supervision et les
// migrations, en plus des connexions réservées au superutilisateur.
const dbConnSafetyMargin = 10

// clampToServerLimit plafonne la taille du pool sous la limite max_connections du serveur
// PostgreSQL : un pool plus grand que ce que le serveur accepte provoque l'erreur
// « too many clients already » (SQLSTATE 53300) dès que la charge sature le pool.
func clampToServerLimit(db *gorm.DB, requested int) int {
	var maxConns, reserved int
	if err := db.Raw("SELECT current_setting('max_connections')::int, current_setting('superuser_reserved_connections')::int").
		Row().Scan(&maxConns, &reserved); err != nil {
		log.Printf("⚠️  Lecture de max_connections impossible, pool conservé à %d: %v", requested, err)
		return requested
	}
	limit := maxConns - reserved - dbConnSafetyMargin
	if limit < 1 {
		limit = 1
	}
	if requested > limit {
		log.Printf("⚠️  DB_MAX_OPEN_CONNS=%d dépasse la capacité PostgreSQL (max_connections=%d, réservées=%d) : pool plafonné à %d",
			requested, maxConns, reserved, limit)
		return limit
	}
	log.Printf("✓ Pool PostgreSQL: %d connexions max (serveur: max_connections=%d)", requested, maxConns)
	return requested
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
