package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/infra/config"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the SQLite database with GORM, runs AutoMigrate, and creates FTS5 virtual table + triggers.
func InitDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent reads
	db.Exec("PRAGMA journal_mode=WAL")

	// AutoMigrate all entities (ShortLink removed — fields merged into URL)
	if err := db.AutoMigrate(
		&entity.URL{},
		&entity.Embedding{},
		&entity.VisitRecord{},
		&entity.LLMLog{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	// Fix: drop any old unique index on short_code (allows multiple empty/NULL values),
	// then create a partial unique index that only enforces uniqueness for non-empty codes.
	db.Exec("DROP INDEX IF EXISTS idx_t_urls_short_code")
	db.Exec("DROP INDEX IF EXISTS uni_t_urls_short_code")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_t_urls_short_code_uniq ON t_urls(short_code) WHERE short_code != ''")

	// One-time data migration: move t_short_links into t_urls
	if err := migrateShortLinks(db); err != nil {
		return nil, fmt.Errorf("migrate short links: %w", err)
	}

	// One-time data migration: convert legacy hex color values to theme keys
	migrateColorThemes(db)

	// Create FTS5 virtual table (rebuilt to include short_code)
	if err := createFTS5(db); err != nil {
		return nil, fmt.Errorf("create FTS5: %w", err)
	}

	log.Println("[DB] Database initialized successfully")
	return db, nil
}

// migrateShortLinks migrates data from the old t_short_links table into t_urls,
// then drops the old table. This is a one-time idempotent migration.
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

	log.Println("[DB] Migrating data from t_short_links to t_urls...")

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
				log.Printf("[DB] Warning: failed to update URL id=%d with short code %s: %v", existing.ID, ol.Code, err)
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
				log.Printf("[DB] Warning: failed to create URL for short code %s: %v", ol.Code, err)
			}
		}
	}

	// Drop the old table
	if err := db.Exec("DROP TABLE IF EXISTS t_short_links").Error; err != nil {
		return fmt.Errorf("drop t_short_links: %w", err)
	}

	log.Printf("[DB] Migrated %d short links into t_urls", len(oldLinks))
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
	// Includes link, title, keywords, description, category, tags, short_code
	// Uses tokenize="unicode61 tokenchars './:@-_'" to keep URL parts as searchable tokens
	fts5DDL := `
		CREATE VIRTUAL TABLE IF NOT EXISTS t_urls_fts USING fts5(
			link, title, keywords, description, category, tags, short_code,
			content=t_urls, content_rowid=id,
			tokenize="unicode61 tokenchars './:@-_'"
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
		// After INSERT: add to FTS
		`CREATE TRIGGER IF NOT EXISTS t_urls_ai AFTER INSERT ON t_urls BEGIN
			INSERT INTO t_urls_fts(rowid, link, title, keywords, description, category, tags, short_code)
			VALUES (new.id, new.link, new.title, new.keywords, new.description, new.category, new.tags, new.short_code);
		END;`,

		// After UPDATE: remove old, add new
		`CREATE TRIGGER IF NOT EXISTS t_urls_au AFTER UPDATE ON t_urls BEGIN
			INSERT INTO t_urls_fts(t_urls_fts, rowid, link, title, keywords, description, category, tags, short_code)
			VALUES('delete', old.id, old.link, old.title, old.keywords, old.description, old.category, old.tags, old.short_code);
			INSERT INTO t_urls_fts(rowid, link, title, keywords, description, category, tags, short_code)
			VALUES (new.id, new.link, new.title, new.keywords, new.description, new.category, new.tags, new.short_code);
		END;`,

		// After DELETE: remove from FTS
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
// Known mappings: #47ff5d→green, #ca4949→red. All other non-theme values are cleared.
// This is idempotent — rows already using theme keys are unaffected.
func migrateColorThemes(db *gorm.DB) {
	validThemes := []string{"", "green", "red", "cyan", "yellow", "purple", "orange", "blue"}

	// Build a NOT IN clause for valid themes
	placeholders := make([]string, len(validThemes))
	for i, t := range validThemes {
		placeholders[i] = fmt.Sprintf("'%s'", t)
	}
	notIn := "(" + joinStrings(placeholders) + ")"

	// Check if any rows need migration
	var count int64
	db.Raw("SELECT COUNT(*) FROM t_urls WHERE color NOT IN " + notIn + " AND deleted_at IS NULL").Scan(&count)
	if count == 0 {
		return
	}

	log.Printf("[DB] Migrating %d color values to theme keys...", count)
	db.Exec("UPDATE t_urls SET color = 'green' WHERE color = '#47ff5d'")
	db.Exec("UPDATE t_urls SET color = 'red' WHERE color = '#ca4949'")
	// Clear any remaining non-theme color values
	db.Exec("UPDATE t_urls SET color = '' WHERE color NOT IN " + notIn)
	log.Println("[DB] Color theme migration complete")
}

// joinStrings joins string slice with commas.
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
