package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"asm/config"
	"asm/routes"
	"asm/service"
	"asm/service/whatsapp"

	"github.com/joho/godotenv"
)

func main() {
	// Set default timezone to GMT+7 (Asia/Bangkok / Asia/Jakarta)
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err == nil {
		time.Local = loc
	}

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Notice: .env file not loaded, using default/system environment variables")
	}

	// Connect to MySQL CSV Database (imp_sampurna on port 3307)
	dbCsv := config.ConnectCsvDB()

	// Connect to MySQL HO Database if needed
	dbHo := config.ConnectHoDB()

	// Inisialisasi WhatsApp Bot (Pure-Go WhatsMeow)
	var waBot *whatsapp.WhatsAppBot
	if dbCsv != nil {
		csvSyncSvc := service.NewCsvSyncService(dbCsv)
		waBot = whatsapp.NewWhatsAppBot(dbCsv, csvSyncSvc)
		go func() {
			if err := waBot.Start(); err != nil {
				log.Printf("Warning: WhatsApp Bot error: %v\n", err)
			}
		}()
		defer waBot.Stop()
	}

	// Setup Router
	r := routes.SetupRouter(dbCsv, dbHo, waBot)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("ASM Server is running on port %s...\n", port)
	fmt.Printf("Access HO Dashboard:   http://localhost:%s/prj-ho\n", port)
	fmt.Printf("Access CSV Dashboard:  http://localhost:%s/prj-csv\n", port)
	fmt.Printf("WhatsApp Scan QR:      http://localhost:%s/wa/qr\n", port)
	fmt.Printf("WhatsApp Bot Status:   http://localhost:%s/wa/status\n", port)

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
