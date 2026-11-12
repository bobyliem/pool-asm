package config

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DBCsv *gorm.DB
	DBHo  *gorm.DB
)

func ConnectCsvDB() *gorm.DB {
	dbUser := os.Getenv("DB_CSV_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_CSV_PASS")
	dbHost := os.Getenv("DB_CSV_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DB_CSV_PORT")
	if dbPort == "" {
		dbPort = "3307"
	}
	dbName := os.Getenv("DB_CSV_NAME")
	if dbName == "" {
		dbName = "imp_sampurna"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})

	if err != nil {
		log.Printf("Warning: Gagal konek ke DB CSV (%s:%s/%s): %v\n", dbHost, dbPort, dbName, err)
		return nil
	}

	fmt.Printf("Successfully connected to MySQL CSV Database (%s:%s/%s)!\n", dbHost, dbPort, dbName)
	DBCsv = db
	return db
}

func ConnectHoDB() *gorm.DB {
	dbUser := os.Getenv("DB_HO_USER")
	dbPass := os.Getenv("DB_HO_PASS")
	dbHost := os.Getenv("DB_HO_HOST")
	dbPort := os.Getenv("DB_HO_PORT")
	dbName := os.Getenv("DB_HO_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})

	if err != nil {
		log.Printf("Warning: Gagal konek ke DB HO (%s:%s/%s): %v\n", dbHost, dbPort, dbName, err)
		return nil
	}

	fmt.Printf("Successfully connected to MySQL HO Database (%s:%s/%s)!\n", dbHost, dbPort, dbName)
	DBHo = db
	return db
}
