package database

import (
	"context"
	"cross-exchange-arbitrage/models"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// InitializeOrderStatuses inserisce i dati iniziali per gli stati degli ordini
func InitializeOrderStatuses(db *gorm.DB) error {
	ctx := context.Background()

	// Stati ordini Bybit V5
	orderStatuses := []models.OrderStatusEntity{
		{
			StatusName:  "New",
			Description: "Ordine piazzato con successo e in attesa di esecuzione",
			IsActive:    true,
		},
		{
			StatusName:  "PartiallyFilled",
			Description: "Ordine parzialmente eseguito",
			IsActive:    true,
		},
		{
			StatusName:  "Untriggered",
			Description: "Ordine condizionale creato ma non ancora attivato",
			IsActive:    true,
		},
		{
			StatusName:  "Rejected",
			Description: "Ordine rifiutato dal sistema",
			IsActive:    true,
		},
		{
			StatusName:  "PartiallyFilledCanceled",
			Description: "Ordine parzialmente eseguito e poi cancellato (solo spot)",
			IsActive:    true,
		},
		{
			StatusName:  "Filled",
			Description: "Ordine completamente eseguito",
			IsActive:    true,
		},
		{
			StatusName:  "Cancelled",
			Description: "Ordine cancellato dall'utente o dal sistema",
			IsActive:    true,
		},
		{
			StatusName:  "Triggered",
			Description: "Ordine condizionale attivato e passato a New",
			IsActive:    true,
		},
		{
			StatusName:  "Deactivated",
			Description: "Ordine disattivato (TP/SL spot, ordini condizionali, OCO)",
			IsActive:    true,
		},
	}

	// Inserimento batch per performance
	for _, status := range orderStatuses {
		// Verifica se lo stato esiste già
		var existingStatus models.OrderStatusEntity
		err := db.WithContext(ctx).Where("status_name = ?", status.StatusName).First(&existingStatus).Error

		if err == gorm.ErrRecordNotFound {
			// Lo stato non esiste, lo crea
			if err := db.WithContext(ctx).Create(&status).Error; err != nil {
				return fmt.Errorf("failed to create order status %s: %w", status.StatusName, err)
			}
			log.Printf("Created order status: %s", status.StatusName)
		} else if err != nil {
			return fmt.Errorf("failed to check existing order status %s: %w", status.StatusName, err)
		} else {
			// Lo stato esiste già, lo aggiorna se necessario
			if existingStatus.Description != status.Description || existingStatus.IsActive != status.IsActive {
				existingStatus.Description = status.Description
				existingStatus.IsActive = status.IsActive
				if err := db.WithContext(ctx).Save(&existingStatus).Error; err != nil {
					return fmt.Errorf("failed to update order status %s: %w", status.StatusName, err)
				}
				log.Printf("Updated order status: %s", status.StatusName)
			}
		}
	}

	log.Println("Order statuses initialized successfully")
	return nil
}

// InitializeDatabaseWithData inizializza il database con connessione, migrazioni e dati iniziali
func InitializeDatabaseWithData(config *Config) (*gorm.DB, error) {
	// Inizializzazione database
	db, err := InitializeDatabase(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Inizializzazione dati
	if err := InitializeOrderStatuses(db); err != nil {
		return nil, fmt.Errorf("failed to initialize order statuses: %w", err)
	}

	return db, nil
}

// SeedTestData inserisce dati di test per sviluppo
func SeedTestData(db *gorm.DB) error {
	ctx := context.Background()

	// Recupera gli stati ordine per i test
	var newStatus, filledStatus models.OrderStatusEntity
	if err := db.WithContext(ctx).Where("status_name = ?", "New").First(&newStatus).Error; err != nil {
		return fmt.Errorf("failed to find New status: %w", err)
	}
	if err := db.WithContext(ctx).Where("status_name = ?", "Filled").First(&filledStatus).Error; err != nil {
		return fmt.Errorf("failed to find Filled status: %w", err)
	}

	// Dati di test
	testOrders := []models.Order{
		{
			OrderID:         "TEST_ORDER_001",
			Symbol:          "BTCUSDT",
			Side:            models.OrderSideTypeBuy,
			OrderPrice:      45000.00000000,
			Quantity:        0.00100000,
			TakeProfitPrice: func() *float64 { tp := 46000.00000000; return &tp }(),
			StopLossPrice:   func() *float64 { sl := 44000.00000000; return &sl }(),
			OrderStatusID:   newStatus.ID,
			Result:          models.OrderResultPending,
			PnL:             0.00000000,
			PnLPercentage:   0.0000,
		},
		{
			OrderID:         "TEST_ORDER_002",
			Symbol:          "ETHUSDT",
			Side:            models.OrderSideTypeSell,
			OrderPrice:      3000.00000000,
			Quantity:        0.1,
			TakeProfitPrice: func() *float64 { tp := 2900.00000000; return &tp }(),
			StopLossPrice:   func() *float64 { sl := 3100.00000000; return &sl }(),
			OrderStatusID:   filledStatus.ID,
			Result:          models.OrderResultProfit,
			PnL:             10.00000000,
			PnLPercentage:   3.3333,
		},
	}

	// Inserimento dati di test
	for _, order := range testOrders {
		// Verifica se l'ordine esiste già
		var existingOrder models.Order
		err := db.WithContext(ctx).Where("order_id = ?", order.OrderID).First(&existingOrder).Error

		if err == gorm.ErrRecordNotFound {
			// L'ordine non esiste, lo crea
			if err := db.WithContext(ctx).Create(&order).Error; err != nil {
				return fmt.Errorf("failed to create test order %s: %w", order.OrderID, err)
			}
			log.Printf("Created test order: %s", order.OrderID)
		} else if err != nil {
			return fmt.Errorf("failed to check existing test order %s: %w", order.OrderID, err)
		}
	}

	log.Println("Test data seeded successfully")
	return nil
}

// CleanupTestData rimuove i dati di test
func CleanupTestData(db *gorm.DB) error {
	ctx := context.Background()

	// Rimuove ordini di test
	if err := db.WithContext(ctx).Where("order_id LIKE ?", "TEST_ORDER_%").Delete(&models.Order{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup test orders: %w", err)
	}

	// Rimuove audit di test
	if err := db.WithContext(ctx).Where("order_id LIKE ?", "TEST_ORDER_%").Delete(&models.OrderAudit{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup test audit records: %w", err)
	}

	log.Println("Test data cleaned up successfully")
	return nil
}
