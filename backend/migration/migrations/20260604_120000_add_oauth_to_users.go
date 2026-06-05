package migrations

import (
	"socialpredict/migration"
	"socialpredict/models"

	"gorm.io/gorm"
)

// MigrateAddOAuthToUsers contains the core logic to add OAuth fields and alter password safety
func MigrateAddOAuthToUsers(db *gorm.DB) error {
	m := db.Migrator()

	// 1) Add auth_provider column if missing (Defaults to 'local')
	if !m.HasColumn(&models.User{}, "AuthProvider") {
		if err := m.AddColumn(&models.User{}, "AuthProvider"); err != nil {
			return err
		}
	}

	// 2) Add auth_id column if missing
	if !m.HasColumn(&models.User{}, "AuthID") {
		if err := m.AddColumn(&models.User{}, "AuthID"); err != nil {
			return err
		}
	}

	// 3) Backfill defaults for existing rows (Ensure all current users are marked 'local')
	if err := db.Model(&models.User{}).
		Where("auth_provider IS NULL OR auth_provider = ''").
		Update("auth_provider", "local").Error; err != nil {
		return err
	}

	// 4) Safely loosen the NOT NULL constraint on the password field.
	// Because SQLite does not support standard ALTER COLUMN modifications cleanly,
	// we use GORM's Dialector check to run raw SQL safely only on heavy engines like PostgreSQL.
	if db.Dialector.Name() == "postgres" {
		err := db.Exec("ALTER TABLE users ALTER COLUMN password DROP NOT NULL;").Error
		if err != nil {
			return err
		}
	}

	return nil
}

// Register the migration with a current timestamp.
func init() {
	migration.Register("20260604120000", func(db *gorm.DB) error {
		return MigrateAddOAuthToUsers(db)
	})
}
