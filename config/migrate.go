package config

import (
	"crypto/md5"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

//go:embed migrations/001_csv_schema.sql
var csvSchemaFS embed.FS

const csvSchemaVersion = "001_csv_schema"

var dbNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// MigrateCsv membuat database CSV kalau belum ada, lalu menjalankan file migrasi
// yang belum tercatat. Tabel yang sudah ada tidak dihapus.
func MigrateCsv() error {
	cfg := loadCsvConfig()
	if !dbNamePattern.MatchString(cfg.name) {
		return fmt.Errorf("nama database tidak valid: %s", cfg.name)
	}

	if err := ensureMysqlUser(cfg); err != nil {
		return err
	}

	dbDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		cfg.user, cfg.pass, cfg.host, cfg.port, cfg.name)
	db, err := sql.Open("mysql", dbDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(64) NOT NULL PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`); err != nil {
		return fmt.Errorf("gagal membuat schema_migrations: %w", err)
	}

	var applied int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, csvSchemaVersion).Scan(&applied); err != nil {
		return err
	}
	if applied == 0 {
		sqlText, err := csvSchemaFS.ReadFile("migrations/001_csv_schema.sql")
		if err != nil {
			return err
		}
		for _, stmt := range splitSQL(string(sqlText)) {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("migrasi %s gagal: %w", csvSchemaVersion, err)
			}
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`, csvSchemaVersion, time.Now()); err != nil {
			return err
		}
	}

	if err := bootstrapAdmin(db); err != nil {
		return err
	}

	fmt.Printf("Database CSV %s siap.\n", cfg.name)
	return nil
}

func ensureMysqlUser(cfg csvConfig) error {
	rootPass := os.Getenv("DB_ROOT_PASS")
	if rootPass == "" {
		rootPass = cfg.pass
	}
	if rootPass == "" {
		return nil
	}

	serverDSN := fmt.Sprintf("root:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true",
		rootPass, cfg.host, cfg.port)
	server, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return err
	}
	defer server.Close()
	if err := server.Ping(); err != nil {
		return fmt.Errorf("tidak bisa konek sebagai root: %w", err)
	}

	createDB := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.name,
	)
	if _, err := server.Exec(createDB); err != nil {
		return fmt.Errorf("gagal membuat database %s: %w", cfg.name, err)
	}

	if cfg.user == "root" || cfg.pass == "" {
		return nil
	}
	if !dbNamePattern.MatchString(cfg.user) {
		return fmt.Errorf("nama user MySQL tidak valid: %s", cfg.user)
	}

	appUser := cfg.user
	quotedPass := "'" + strings.ReplaceAll(cfg.pass, "'", "''") + "'"
	for _, host := range []string{"%", "localhost"} {
		createUser := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY %s", appUser, host, quotedPass)
		if _, err := server.Exec(createUser); err != nil {
			return fmt.Errorf("gagal membuat user MySQL %s@%s: %w", appUser, host, err)
		}
		alterUser := fmt.Sprintf("ALTER USER '%s'@'%s' IDENTIFIED BY %s", appUser, host, quotedPass)
		if _, err := server.Exec(alterUser); err != nil {
			return fmt.Errorf("gagal mengatur password user MySQL %s@%s: %w", appUser, host, err)
		}
		grant := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%s'", cfg.name, appUser, host)
		if _, err := server.Exec(grant); err != nil {
			return fmt.Errorf("gagal memberi hak user MySQL %s@%s: %w", appUser, host, err)
		}
	}
	if _, err := server.Exec("FLUSH PRIVILEGES"); err != nil {
		return err
	}
	fmt.Printf("User MySQL %s siap untuk database %s.\n", appUser, cfg.name)
	return nil
}

func bootstrapAdmin(db *sql.DB) error {
	username := strings.TrimSpace(os.Getenv("ASM_BOOTSTRAP_USER"))
	password := os.Getenv("ASM_BOOTSTRAP_PASS")
	if username == "" || password == "" {
		return nil
	}

	sum := md5.Sum([]byte(password))
	hash := hex.EncodeToString(sum[:])
	_, err := db.Exec(`INSERT INTO menu_login (usrn, pass, nama, aktif, grp_id, createdby, created)
		SELECT ?, ?, ?, 1, 1, 'migrate', NOW()
		WHERE NOT EXISTS (SELECT 1 FROM menu_login WHERE usrn = ?)`,
		username, hash, username, username)
	if err != nil {
		return fmt.Errorf("gagal membuat user awal: %w", err)
	}
	return nil
}

func splitSQL(script string) []string {
	var b strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	var stmts []string
	for _, part := range strings.Split(b.String(), ";") {
		stmt := strings.TrimSpace(part)
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}
