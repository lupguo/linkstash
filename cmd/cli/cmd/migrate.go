package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/spf13/cobra"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	migrateSQLitePath string
	migrateMySQLDSN   string
	migrateMySQLUser  string
	migrateMySQLPass  string
	migrateMySQLHost  string
	migrateMySQLPort  int
	migrateMySQLDB    string
	migrateBatchSize  int
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate data between SQLite and MySQL databases",
	Long: `Migrate all data from a SQLite database to a MySQL database.

Examples:
  # Using DSN string
  linkstash migrate --sqlite-path ./data/linkstash.db \
    --mysql-dsn "root:Secret123.@tcp(127.0.0.1:3308)/linkstash_db?charset=utf8mb4&parseTime=True&loc=Local"

  # Using individual parameters
  linkstash migrate --sqlite-path ./data/linkstash.db \
    --mysql-user root --mysql-password Secret123. \
    --mysql-host 127.0.0.1 --mysql-port 3308 --mysql-db linkstash_db`,
	// Override PersistentPreRun to skip auth (migrate doesn't need a running server)
	PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	RunE:             runMigrate,
}

func init() {
	migrateCmd.Flags().StringVar(&migrateSQLitePath, "sqlite-path", "./data/linkstash.db", "Path to SQLite database file")
	migrateCmd.Flags().StringVar(&migrateMySQLDSN, "mysql-dsn", "", "MySQL DSN (overrides individual mysql-* flags)")
	migrateCmd.Flags().StringVar(&migrateMySQLUser, "mysql-user", "root", "MySQL user")
	migrateCmd.Flags().StringVar(&migrateMySQLPass, "mysql-password", "", "MySQL password")
	migrateCmd.Flags().StringVar(&migrateMySQLHost, "mysql-host", "127.0.0.1", "MySQL host")
	migrateCmd.Flags().IntVar(&migrateMySQLPort, "mysql-port", 3306, "MySQL port")
	migrateCmd.Flags().StringVar(&migrateMySQLDB, "mysql-db", "linkstash_db", "MySQL database name")
	migrateCmd.Flags().IntVar(&migrateBatchSize, "batch-size", 100, "Number of records per batch insert")

	RootCmd.AddCommand(migrateCmd)
}

func buildMySQLDSN() string {
	if migrateMySQLDSN != "" {
		return migrateMySQLDSN
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		migrateMySQLUser, migrateMySQLPass, migrateMySQLHost, migrateMySQLPort, migrateMySQLDB)
}

// buildMySQLMigrationDSN returns a DSN for data migration that tolerates non-UTF8 data.
func buildMySQLMigrationDSN() string {
	if migrateMySQLDSN != "" {
		return migrateMySQLDSN
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=binary&parseTime=True&loc=Local",
		migrateMySQLUser, migrateMySQLPass, migrateMySQLHost, migrateMySQLPort, migrateMySQLDB)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	// Validate SQLite path exists
	if _, err := os.Stat(migrateSQLitePath); os.IsNotExist(err) {
		return fmt.Errorf("sqlite database not found: %s", migrateSQLitePath)
	}

	dsn := buildMySQLDSN()
	migrationDSN := buildMySQLMigrationDSN()
	fmt.Printf("Migration: SQLite(%s) → MySQL(%s)\n", migrateSQLitePath, migrateMySQLDB)

	// Open source SQLite
	srcDB, err := gorm.Open(sqlite.Open(migrateSQLitePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	srcDB.Exec("PRAGMA journal_mode=WAL")

	// Open target MySQL for schema creation (utf8mb4)
	schemaDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("open mysql (schema): %w", err)
	}

	// AutoMigrate on target to ensure tables exist
	fmt.Println("Creating/verifying MySQL tables...")
	if err := schemaDB.AutoMigrate(
		&entity.URL{},
		&entity.Embedding{},
		&entity.VisitRecord{},
		&entity.LLMLog{},
	); err != nil {
		return fmt.Errorf("auto migrate target: %w", err)
	}

	// Open target MySQL for data migration (binary charset to handle non-UTF8 data)
	dstDB, err := gorm.Open(mysql.Open(migrationDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("open mysql (migration): %w", err)
	}

	// Disable strict mode to allow inserting data with non-UTF8 bytes
	dstDB.Exec("SET SESSION sql_mode = ''")

	// Migrate each table
	tables := []struct {
		name    string
		migrate func(src, dst *gorm.DB) (int, error)
	}{
		{"t_urls", migrateURLs},
		{"t_embeddings", migrateEmbeddings},
		{"t_visit_records", migrateVisitRecords},
		{"t_llm_logs", migrateLLMLogs},
	}

	for _, t := range tables {
		count, err := t.migrate(srcDB, dstDB)
		if err != nil {
			slog.Error("migration failed", "table", t.name, "error", err)
			return fmt.Errorf("migrate %s: %w", t.name, err)
		}
		fmt.Printf("  ✓ %s: %d records migrated\n", t.name, count)
	}

	fmt.Println("\nMigration completed successfully!")
	return nil
}

func migrateURLs(src, dst *gorm.DB) (int, error) {
	var urls []entity.URL
	// Use Unscoped to include soft-deleted records
	if err := src.Unscoped().Find(&urls).Error; err != nil {
		return 0, fmt.Errorf("read source: %w", err)
	}
	if len(urls) == 0 {
		return 0, nil
	}

	// Batch insert
	for i := 0; i < len(urls); i += migrateBatchSize {
		end := i + migrateBatchSize
		if end > len(urls) {
			end = len(urls)
		}
		batch := urls[i:end]
		if err := dst.Create(&batch).Error; err != nil {
			return 0, fmt.Errorf("insert batch at offset %d: %w", i, err)
		}
	}
	return len(urls), nil
}

func migrateEmbeddings(src, dst *gorm.DB) (int, error) {
	var embeddings []entity.Embedding
	if err := src.Find(&embeddings).Error; err != nil {
		return 0, fmt.Errorf("read source: %w", err)
	}
	if len(embeddings) == 0 {
		return 0, nil
	}

	for i := 0; i < len(embeddings); i += migrateBatchSize {
		end := i + migrateBatchSize
		if end > len(embeddings) {
			end = len(embeddings)
		}
		batch := embeddings[i:end]
		if err := dst.Create(&batch).Error; err != nil {
			return 0, fmt.Errorf("insert batch at offset %d: %w", i, err)
		}
	}
	return len(embeddings), nil
}

func migrateVisitRecords(src, dst *gorm.DB) (int, error) {
	var records []entity.VisitRecord
	if err := src.Find(&records).Error; err != nil {
		return 0, fmt.Errorf("read source: %w", err)
	}
	if len(records) == 0 {
		return 0, nil
	}

	for i := 0; i < len(records); i += migrateBatchSize {
		end := i + migrateBatchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]
		if err := dst.Create(&batch).Error; err != nil {
			return 0, fmt.Errorf("insert batch at offset %d: %w", i, err)
		}
	}
	return len(records), nil
}

func migrateLLMLogs(src, dst *gorm.DB) (int, error) {
	var logs []entity.LLMLog
	if err := src.Find(&logs).Error; err != nil {
		return 0, fmt.Errorf("read source: %w", err)
	}
	if len(logs) == 0 {
		return 0, nil
	}

	for i := 0; i < len(logs); i += migrateBatchSize {
		end := i + migrateBatchSize
		if end > len(logs) {
			end = len(logs)
		}
		batch := logs[i:end]
		if err := dst.Create(&batch).Error; err != nil {
			return 0, fmt.Errorf("insert batch at offset %d: %w", i, err)
		}
	}
	return len(logs), nil
}
