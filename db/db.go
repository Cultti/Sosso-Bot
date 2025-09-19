package db

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var database *gorm.DB

func Init(dbPath string, models ...interface{}) {
	var err error
	database, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	if err := database.AutoMigrate(&Subscription{}); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}
}

// Expose a safe getter (optional, if you sometimes want raw GORM)
func DB() *gorm.DB { return database }
