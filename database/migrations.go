package database

import (
	"context"
	"cross-exchange-arbitrage/models"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config rappresenta la configurazione del database
type Config struct {
	FilePath string // Percorso del file SQLite
}

// DefaultConfig restituisce una configurazione di default
func DefaultConfig() *Config {
	return &Config{
		FilePath: getEnv("DB_FILE_PATH", "./trading_bot.db"),
	}
}

// getEnv restituisce il valore di una variabile d'ambiente o un valore di default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Connect stabilisce una connessione al database SQLite
func Connect(config *Config) (*gorm.DB, error) {
	// Configurazione logger GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(sqlite.Open(config.FilePath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configurazione connection pool per SQLite
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configurazione connection pool ottimizzata per SQLite
	sqlDB.SetMaxIdleConns(1)    // SQLite funziona meglio con poche connessioni
	sqlDB.SetMaxOpenConns(1)    // Una sola connessione per SQLite
	sqlDB.SetConnMaxLifetime(0) // Nessun timeout per SQLite

	return db, nil
}

// Migrate esegue le migrazioni per creare le tabelle
func Migrate(db *gorm.DB) error {
	// Auto-migrazione per creare le tabelle (ordine importante per foreign key)
	err := db.AutoMigrate(
		&models.OrderStatusEntity{},
		&models.Order{},
		&models.OrderAudit{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Creazione indici aggiuntivi per performance
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Creazione vincoli CHECK per business logic
	if err := createConstraints(db); err != nil {
		return fmt.Errorf("failed to create constraints: %w", err)
	}

	return nil
}

// createIndexes crea indici aggiuntivi per performance
func createIndexes(db *gorm.DB) error {
	// Indici compositi per query frequenti
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_symbol_status ON orders (symbol, order_status_id);",
		"CREATE INDEX IF NOT EXISTS idx_symbol_result ON orders (symbol, result);",
		"CREATE INDEX IF NOT EXISTS idx_created_status ON orders (created_at, order_status_id);",
		"CREATE INDEX IF NOT EXISTS idx_orders_compound ON orders (symbol, side, result, created_at);",
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	return nil
}

// createConstraints crea vincoli CHECK per business logic (SQLite compatibile)
func createConstraints(db *gorm.DB) error {
	constraints := []string{
		// Vincolo per take profit (Buy: TP > OrderPrice, Sell: TP < OrderPrice)
		`CREATE TRIGGER IF NOT EXISTS chk_take_profit_buy 
		 BEFORE INSERT ON orders
		 BEGIN
		   SELECT CASE
		     WHEN (NEW.side = 'Buy' AND NEW.take_profit_price IS NOT NULL AND NEW.take_profit_price <= NEW.order_price) OR
		          (NEW.side = 'Sell' AND NEW.take_profit_price IS NOT NULL AND NEW.take_profit_price >= NEW.order_price)
		     THEN RAISE(ABORT, 'Invalid take profit price for order side')
		   END;
		 END;`,

		// Vincolo per stop loss (Buy: SL < OrderPrice, Sell: SL > OrderPrice)
		`CREATE TRIGGER IF NOT EXISTS chk_stop_loss_buy 
		 BEFORE INSERT ON orders
		 BEGIN
		   SELECT CASE
		     WHEN (NEW.side = 'Buy' AND NEW.stop_loss_price IS NOT NULL AND NEW.stop_loss_price >= NEW.order_price) OR
		          (NEW.side = 'Sell' AND NEW.stop_loss_price IS NOT NULL AND NEW.stop_loss_price <= NEW.order_price)
		     THEN RAISE(ABORT, 'Invalid stop loss price for order side')
		   END;
		 END;`,

		// Vincolo per prezzi positivi
		`CREATE TRIGGER IF NOT EXISTS chk_positive_prices 
		 BEFORE INSERT ON orders
		 BEGIN
		   SELECT CASE
		     WHEN NEW.order_price <= 0 OR NEW.quantity <= 0 OR
		          (NEW.take_profit_price IS NOT NULL AND NEW.take_profit_price <= 0) OR
		          (NEW.stop_loss_price IS NOT NULL AND NEW.stop_loss_price <= 0)
		     THEN RAISE(ABORT, 'Prices and quantities must be positive')
		   END;
		 END;`,

		// Vincolo per side valido
		`CREATE TRIGGER IF NOT EXISTS chk_valid_side 
		 BEFORE INSERT ON orders
		 BEGIN
		   SELECT CASE
		     WHEN NEW.side NOT IN ('Buy', 'Sell')
		     THEN RAISE(ABORT, 'Invalid order side')
		   END;
		 END;`,

		// Vincolo per result valido
		`CREATE TRIGGER IF NOT EXISTS chk_valid_result 
		 BEFORE INSERT ON orders
		 BEGIN
		   SELECT CASE
		     WHEN NEW.result NOT IN ('Profit', 'Loss', 'Pending')
		     THEN RAISE(ABORT, 'Invalid order result')
		   END;
		 END;`,
	}

	for _, constraintSQL := range constraints {
		if err := db.Exec(constraintSQL).Error; err != nil {
			log.Printf("Warning: failed to create constraint: %v", err)
		}
	}

	return nil
}

// InitializeDatabase inizializza il database con connessione e migrazioni
func InitializeDatabase(config *Config) (*gorm.DB, error) {
	// Connessione al database
	db, err := Connect(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test della connessione
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Esecuzione migrazioni
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}

// Close chiude la connessione al database
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck verifica lo stato del database
func HealthCheck(db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}
