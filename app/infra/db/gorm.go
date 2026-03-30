package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/infra/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the database with GORM based on the configured driver (sqlite or mysql).
// It runs AutoMigrate and driver-specific setup (FTS5 for SQLite, index fixes for MySQL).
func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	var database *gorm.DB
	var err error

	switch {
	case cfg.IsMySQL():
		database, err = initMySQL(cfg)
	default:
		database, err = initSQLite(cfg)
	}
	if err != nil {
		return nil, err
	}

	// AutoMigrate all entities (shared across both drivers)
	if err := database.AutoMigrate(
		&entity.URL{},
		&entity.Embedding{},
		&entity.VisitRecord{},
		&entity.LLMLog{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	// Driver-specific post-migration setup
	if cfg.IsSQLite() {
		if err := postMigrateSQLite(database); err != nil {
			return nil, err
		}
	} else if cfg.IsMySQL() {
		if err := postMigrateMySQL(database); err != nil {
			return nil, err
		}
	}

	slog.Info("database initialized successfully", "component", "db", "driver", cfg.Driver)
	return database, nil
}

// initSQLite opens a SQLite database and enables WAL mode.
func initSQLite(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dbPath := cfg.GetSQLitePath()

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// Enable WAL mode for better concurrent reads
	db.Exec("PRAGMA journal_mode=WAL")

	slog.Info("sqlite database opened", "component", "db", "path", dbPath)
	return db, nil
}

// initMySQL opens a MySQL database connection.
func initMySQL(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := cfg.MySQL.DSN()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open mysql database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MySQL.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MySQL.MaxIdleConns)

	slog.Info("mysql database opened", "component", "db", "host", cfg.MySQL.Host, "port", cfg.MySQL.Port, "dbname", cfg.MySQL.DBName)
	return db, nil
}

// postMigrateSQLite runs SQLite-specific setup after AutoMigrate:
// partial unique index, legacy migrations, and FTS5 virtual table.
func postMigrateSQLite(db *gorm.DB) error {
	// Fix: drop any old unique index on short_code (allows multiple empty/NULL values),
	// then create a partial unique index that only enforces uniqueness for non-empty codes.
	db.Exec("DROP INDEX IF EXISTS idx_t_urls_short_code")
	db.Exec("DROP INDEX IF EXISTS uni_t_urls_short_code")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_t_urls_short_code_uniq ON t_urls(short_code) WHERE short_code != ''")

	// One-time data migration: move t_short_links into t_urls
	if err := migrateShortLinks(db); err != nil {
		return fmt.Errorf("migrate short links: %w", err)
	}

	// One-time data migration: convert legacy hex color values to theme keys
	migrateColorThemes(db)

	// Create FTS5 virtual table (rebuilt to include short_code)
	if err := createFTS5(db); err != nil {
		return fmt.Errorf("create FTS5: %w", err)
	}

	return nil
}

// postMigrateMySQL runs MySQL-specific setup after AutoMigrate.
func postMigrateMySQL(db *gorm.DB) error {
	// For MySQL, we can't use partial unique index.
	// Instead, ensure short_code allows empty string and use application-level uniqueness.
	// GORM AutoMigrate already handles the basic index from struct tags.

	// One-time data migration: convert legacy hex color values to theme keys
	migrateColorThemes(db)

	// Reorder columns: move created_at, updated_at, deleted_at to the end of each table
	reorderStatements := []string{
		// t_urls
		"ALTER TABLE t_urls MODIFY COLUMN created_at datetime(3) COMMENT '创建时间' AFTER favicon",
		"ALTER TABLE t_urls MODIFY COLUMN updated_at datetime(3) COMMENT '更新时间' AFTER created_at",
		"ALTER TABLE t_urls MODIFY COLUMN deleted_at datetime(3) COMMENT '删除时间(软删除)' AFTER updated_at",
		// t_embeddings
		"ALTER TABLE t_embeddings MODIFY COLUMN created_at datetime(3) COMMENT '创建时间' AFTER vector",
		"ALTER TABLE t_embeddings MODIFY COLUMN updated_at datetime(3) COMMENT '更新时间' AFTER created_at",
		"ALTER TABLE t_embeddings MODIFY COLUMN deleted_at datetime(3) COMMENT '删除时间(软删除)' AFTER updated_at",
		// t_visit_records
		"ALTER TABLE t_visit_records MODIFY COLUMN created_at datetime(3) COMMENT '创建时间' AFTER user_agent",
		"ALTER TABLE t_visit_records MODIFY COLUMN updated_at datetime(3) COMMENT '更新时间' AFTER created_at",
		"ALTER TABLE t_visit_records MODIFY COLUMN deleted_at datetime(3) COMMENT '删除时间(软删除)' AFTER updated_at",
		// t_llm_logs
		"ALTER TABLE t_llm_logs MODIFY COLUMN created_at datetime(3) COMMENT '创建时间' AFTER success",
		"ALTER TABLE t_llm_logs MODIFY COLUMN updated_at datetime(3) COMMENT '更新时间' AFTER created_at",
		"ALTER TABLE t_llm_logs MODIFY COLUMN deleted_at datetime(3) COMMENT '删除时间(软删除)' AFTER updated_at",
	}
	for _, stmt := range reorderStatements {
		if err := db.Exec(stmt).Error; err != nil {
			slog.Warn("column reorder failed (non-fatal)", "component", "db", "stmt", stmt, "error", err)
		}
	}

	return nil
}

// migrateShortLinks migrates data from the old t_short_links table into t_urls,
// then drops the old table. This is a one-time idempotent migration (SQLite only).
func migrateShortLinks(db *gorm.DB) error {
	// Check if the old table exists
	var count int64
	err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='t_short_links'").Scan(&count).Error
	if err != nil {
		return fmt.Errorf("check t_short_links existence: %w", err)
	}
	if count == 0 {
		return nil // Old table doesn't exist; nothing to migrate
	}

	slog.Info("migrating data from t_short_links to t_urls", "component", "db")

	// Read all rows from the old table
	type oldShortLink struct {
		Code       string
		LongURL    string  `gorm:"column:long_url"`
		ExpiresAt  *string `gorm:"column:expires_at"`
		ClickCount int     `gorm:"column:click_count"`
	}
	var oldLinks []oldShortLink
	if err := db.Raw("SELECT code, long_url, expires_at, click_count FROM t_short_links WHERE deleted_at IS NULL").Scan(&oldLinks).Error; err != nil {
		return fmt.Errorf("read t_short_links: %w", err)
	}

	for _, ol := range oldLinks {
		// Try to find a matching URL by link
		var existing entity.URL
		err := db.Where("link = ?", ol.LongURL).First(&existing).Error
		if err == nil {
			// Found — update with short code data
			updates := map[string]interface{}{
				"short_code":  ol.Code,
				"visit_count": gorm.Expr("visit_count + ?", ol.ClickCount),
			}
			if ol.ExpiresAt != nil {
				updates["short_expires_at"] = *ol.ExpiresAt
			}
			if err := db.Model(&entity.URL{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
				slog.Warn("failed to update URL with short code", "component", "db", "url_id", existing.ID, "short_code", ol.Code, "error", err)
			}
		} else {
			// Not found — create a new URL record
			newURL := entity.URL{
				Link:       ol.LongURL,
				ShortCode:  ol.Code,
				VisitCount: ol.ClickCount,
				Status:     "pending",
			}
			if err := db.Create(&newURL).Error; err != nil {
				slog.Warn("failed to create URL for short code", "component", "db", "short_code", ol.Code, "error", err)
			}
		}
	}

	// Drop the old table
	if err := db.Exec("DROP TABLE IF EXISTS t_short_links").Error; err != nil {
		return fmt.Errorf("drop t_short_links: %w", err)
	}

	slog.Info("migrated short links into t_urls", "component", "db", "count", len(oldLinks))
	return nil
}

func createFTS5(db *gorm.DB) error {
	// Drop old triggers and FTS table to rebuild with short_code column
	dropStatements := []string{
		"DROP TRIGGER IF EXISTS t_urls_ai",
		"DROP TRIGGER IF EXISTS t_urls_au",
		"DROP TRIGGER IF EXISTS t_urls_ad",
		"DROP TABLE IF EXISTS t_urls_fts",
	}
	for _, stmt := range dropStatements {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("drop old FTS objects: %w", err)
		}
	}

	// Create FTS5 virtual table for full-text search on URLs
	fts5DDL := `
		CREATE VIRTUAL TABLE IF NOT EXISTS t_urls_fts USING fts5(
			link, title, keywords, description, category, tags, short_code,
			content=t_urls, content_rowid=id,
			tokenize="unicode61"
		);
	`
	if err := db.Exec(fts5DDL).Error; err != nil {
		return fmt.Errorf("create FTS5 table: %w", err)
	}

	// Re-populate FTS data from existing rows
	populateSQL := `
		INSERT INTO t_urls_fts(rowid, link, title, keywords, description, category, tags, short_code)
		SELECT id, link, title, keywords, description, category, tags, short_code FROM t_urls WHERE deleted_at IS NULL;
	`
	if err := db.Exec(populateSQL).Error; err != nil {
		return fmt.Errorf("populate FTS5 data: %w", err)
	}

	// Triggers to keep FTS5 in sync with t_urls
	triggers := []string{
		`CREATE TRIGGER IF NOT EXISTS t_urls_ai AFTER INSERT ON t_urls BEGIN
			INSERT INTO t_urls_fts(rowid, link, title, keywords, description, category, tags, short_code)
			VALUES (new.id, new.link, new.title, new.keywords, new.description, new.category, new.tags, new.short_code);
		END;`,

		`CREATE TRIGGER IF NOT EXISTS t_urls_au AFTER UPDATE ON t_urls BEGIN
			INSERT INTO t_urls_fts(t_urls_fts, rowid, link, title, keywords, description, category, tags, short_code)
			VALUES('delete', old.id, old.link, old.title, old.keywords, old.description, old.category, old.tags, old.short_code);
			INSERT INTO t_urls_fts(rowid, link, title, keywords, description, category, tags, short_code)
			VALUES (new.id, new.link, new.title, new.keywords, new.description, new.category, new.tags, new.short_code);
		END;`,

		`CREATE TRIGGER IF NOT EXISTS t_urls_ad AFTER DELETE ON t_urls BEGIN
			INSERT INTO t_urls_fts(t_urls_fts, rowid, link, title, keywords, description, category, tags, short_code)
			VALUES('delete', old.id, old.link, old.title, old.keywords, old.description, old.category, old.tags, old.short_code);
		END;`,
	}

	for _, trigger := range triggers {
		if err := db.Exec(trigger).Error; err != nil {
			return fmt.Errorf("create trigger: %w", err)
		}
	}

	return nil
}

// migrateColorThemes converts legacy hex color values to preset theme keys.
func migrateColorThemes(db *gorm.DB) {
	validThemes := []string{"", "green", "red", "cyan", "yellow", "purple", "orange", "blue"}

	placeholders := make([]string, len(validThemes))
	for i, t := range validThemes {
		placeholders[i] = fmt.Sprintf("'%s'", t)
	}
	notIn := "(" + joinStrings(placeholders) + ")"

	var count int64
	db.Raw("SELECT COUNT(*) FROM t_urls WHERE color NOT IN " + notIn + " AND deleted_at IS NULL").Scan(&count)
	if count == 0 {
		return
	}

	slog.Info("migrating color values to theme keys", "component", "db", "count", count)
	db.Exec("UPDATE t_urls SET color = 'green' WHERE color = '#47ff5d'")
	db.Exec("UPDATE t_urls SET color = 'red' WHERE color = '#ca4949'")
	db.Exec("UPDATE t_urls SET color = '' WHERE color NOT IN " + notIn)
	slog.Info("color theme migration complete", "component", "db")
}

func joinStrings(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ","
		}
		result += v
	}
	return result
}
