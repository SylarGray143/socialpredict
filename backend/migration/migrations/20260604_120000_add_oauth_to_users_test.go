package migrations_test

import (
	"testing"

	"socialpredict/migration/migrations"
	"socialpredict/models"
	"socialpredict/models/modelstesting"

	"gorm.io/gorm"
)

// UserV1 mirrors the old schema configuration before the OAuth changes.
// It forces the table name matching trick so GORM creates the right environment.
type UserV1 struct {
	ID                 int64  `gorm:"primaryKey"`
	Username           string `gorm:"unique;not null"`
	DisplayName        string `gorm:"unique;not null"`
	UserType           string `gorm:"not null"`
	Email              string `gorm:"unique;not null"`
	Password           string `gorm:"not null"` // Hard v1 constraint
	MustChangePassword bool   `gorm:"default:true"`
}

func (UserV1) TableName() string { return "users" }

func seedPreMigrationUser(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	u := UserV1{
		ID:                 1,
		Username:           "testadmin",
		DisplayName:        "Test Admin",
		UserType:           "admin",
		Email:              "admin@local.test",
		Password:           "$2a$14$somefakebcryptstringhash", // Simulate an existing user
		MustChangePassword: true,
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("failed to seed legacy v1 user: %v", err)
	}
	return u.ID
}

func TestMigrateAddOAuthToUsers_AddsColumnsAndBackfillsLocal(t *testing.T) {
	// 1. Arrange: Initialize a fresh in-memory SQLite database instance
	db := modelstesting.NewFakeDB(t)

	// Simulate old V1 state by dropping the new OAuth tracking columns if they exist
	_ = db.Migrator().DropColumn(&models.User{}, "AuthProvider")
	_ = db.Migrator().DropColumn(&models.User{}, "AuthID")

	// Seed the database with a user who registered prior to our architectural change
	userID := seedPreMigrationUser(t, db)

	// 2. Act: Run your newly written migration logic
	if err := migrations.MigrateAddOAuthToUsers(db); err != nil {
		t.Fatalf("migration execution routine aborted with error: %v", err)
	}

	// 3. Assert: Verify the physical columns were appended to the tables
	mig := db.Migrator()
	if !mig.HasColumn(&models.User{}, "AuthProvider") {
		t.Fatalf("expected auth_provider column to be registered inside the users schema layout")
	}
	if !mig.HasColumn(&models.User{}, "AuthID") {
		t.Fatalf("expected auth_id column to be registered inside the users schema layout")
	}

	// Assert: Load the mutated user using the modern structural domain definition
	var mutatedUser models.User
	if err := db.First(&mutatedUser, userID).Error; err != nil {
		t.Fatalf("failed to reload mutated user payload from database context: %v", err)
	}

	// Verify the data backfill accurately converted empty records to local tracking strings
	if mutatedUser.AuthProvider != "local" {
		t.Fatalf("expected legacy user auth provider backfill to resolve as 'local', instead observed: %q", mutatedUser.AuthProvider)
	}

	// Ensure the legacy password payload data survived the structural database updates intact
	if mutatedUser.Password != "$2a$14$somefakebcryptstringhash" {
		t.Fatalf("regression breakdown detected: legacy password token value altered during migration transaction")
	}
}
