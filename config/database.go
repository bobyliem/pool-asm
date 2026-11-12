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

type csvConfig struct {
	user string
	pass string
	host string
	port string
	name string
}

func loadCsvConfig() csvConfig {
	cfg := csvConfig{
		user: os.Getenv("DB_CSV_USER"),
		pass: os.Getenv("DB_CSV_PASS"),
		host: os.Getenv("DB_CSV_HOST"),
		port: os.Getenv("DB_CSV_PORT"),
		name: os.Getenv("DB_CSV_NAME"),
	}
	if cfg.user == "" {
		cfg.user = "root"
	}
	if cfg.host == "" {
		cfg.host = "127.0.0.1"
	}
	if cfg.port == "" {
		cfg.port = "3307"
	}
	if cfg.name == "" {
		cfg.name = "imp_sampurna"
	}
	return cfg
}

func ConnectCsvDB() *gorm.DB {
	cfg := loadCsvConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.user, cfg.pass, cfg.host, cfg.port, cfg.name)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})

	if err != nil {
		log.Printf("Warning: Gagal konek ke DB CSV (%s:%s/%s): %v\n", cfg.host, cfg.port, cfg.name, err)
		return nil
	}

	fmt.Printf("Successfully connected to MySQL CSV Database (%s:%s/%s)!\n", cfg.host, cfg.port, cfg.name)
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
