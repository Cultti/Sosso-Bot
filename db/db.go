package db

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var database *gorm.DB

func Init(dbPath string, models ...interface{}) {
	// Ensure folder exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

	// Open database
	var err error
	database, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Auto-migrate models
	if err := database.AutoMigrate(&Subscription{}); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}
}

// Expose a safe getter (optional, if you sometimes want raw GORM)
func DB() *gorm.DB { return database }
