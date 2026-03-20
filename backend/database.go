package main

import (
	"clinic-system/backend/models" 
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres" // Change this from sqlite to postgres
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	// 1. Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 2. Get the connection string from .env
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("DB_URL is not set in .env file")
	}

	// 3. Open connection using the POSTGRES driver
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 4. Sync your Go models with Supabase tables
	err = database.AutoMigrate(&models.Patient{}, &models.MedicalRecord{}, &models.ParsedItem{}, &models.PriceList{})
	if err != nil {
		log.Printf("Migration warning: %v (continuing anyway)", err)
	}

	DB = database
	log.Println("✅ Database connection successful!")
}