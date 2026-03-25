package repositories

import (
	"github.com/andreiOpran/licenta/operational-node/internal/database"
	"github.com/andreiOpran/licenta/operational-node/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB initializes a fresh in-memory database for each test
func setupTestDB() (*gorm.DB, func()) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	database.DB = db

	// run migrations
	database.DB.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.ActionToken{},
		&models.Wallet{},
	)

	// return the db instance and a cleanup function
	return db, func() {
		database.DB = nil
	}
}
