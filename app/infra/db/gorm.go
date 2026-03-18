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

	// AutoMigrate all entities
	if err := db.AutoMigrate(
		&entity.URL{},
		&entity.Embedding{},
		&entity.ShortLink{},
		&entity.VisitRecord{},
		&entity.LLMLog{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	// Create FTS5 virtual table (idempotent)
	if err := createFTS5(db); err != nil {
		return nil, fmt.Errorf("create FTS5: %w", err)
	}

	log.Println("[DB] Database initialized successfully")
	return db, nil
}

func createFTS5(db *gorm.DB) error {
	// Create FTS5 virtual table for full-text search on URLs
	fts5DDL := `
		CREATE VIRTUAL TABLE IF NOT EXISTS t_urls_fts USING fts5(
			title, keywords, description,
			content=t_urls, content_rowid=id
		);
	`
	if err := db.Exec(fts5DDL).Error; err != nil {
		return fmt.Errorf("create FTS5 table: %w", err)
	}

	// Triggers to keep FTS5 in sync with t_urls
	triggers := []string{
		// After INSERT: add to FTS
		`CREATE TRIGGER IF NOT EXISTS t_urls_ai AFTER INSERT ON t_urls BEGIN
			INSERT INTO t_urls_fts(rowid, title, keywords, description)
			VALUES (new.id, new.title, new.keywords, new.description);
		END;`,

		// After UPDATE: remove old, add new
		`CREATE TRIGGER IF NOT EXISTS t_urls_au AFTER UPDATE ON t_urls BEGIN
			INSERT INTO t_urls_fts(t_urls_fts, rowid, title, keywords, description)
			VALUES('delete', old.id, old.title, old.keywords, old.description);
			INSERT INTO t_urls_fts(rowid, title, keywords, description)
			VALUES (new.id, new.title, new.keywords, new.description);
		END;`,

		// After DELETE: remove from FTS
		`CREATE TRIGGER IF NOT EXISTS t_urls_ad AFTER DELETE ON t_urls BEGIN
			INSERT INTO t_urls_fts(t_urls_fts, rowid, title, keywords, description)
			VALUES('delete', old.id, old.title, old.keywords, old.description);
		END;`,
	}

	for _, trigger := range triggers {
		if err := db.Exec(trigger).Error; err != nil {
			return fmt.Errorf("create trigger: %w", err)
		}
	}

	return nil
}
