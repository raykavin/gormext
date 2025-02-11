// gorm_test.go
package gormext

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// =======================
// Dummy implementation for IRepository
// =======================

type DummyRepo struct{}

func (d *DummyRepo) WithTransaction(fn func(tx IRepository) error) error { return fn(d) }
func (d *DummyRepo) WithContext(ctx context.Context) IRepository         { return d }
func (d *DummyRepo) FirstByID(id any, dest any) error                    { return nil }
func (d *DummyRepo) First(dest any, conds ...any) error                  { return nil }
func (d *DummyRepo) Find(dest any) error                                 { return nil }
func (d *DummyRepo) Create(entity any) error                             { return nil }
func (d *DummyRepo) Update(entity any) error                             { return nil }
func (d *DummyRepo) Delete(entity any) error                             { return nil }
func (d *DummyRepo) Exec(sql string, value ...any) error                 { return nil }
func (d *DummyRepo) IDEqual(id any) IRepository                          { return d }
func (d *DummyRepo) IDIn(ids []any) IRepository                          { return d }
func (d *DummyRepo) Where(query any, args ...any) IRepository            { return d }
func (d *DummyRepo) Joins(query string, args ...any) IRepository         { return d }
func (d *DummyRepo) Preload(query string, args ...any) IRepository       { return d }
func (d *DummyRepo) Order(value any) IRepository                         { return d }
func (d *DummyRepo) IsActive() IRepository                               { return d }
func (d *DummyRepo) Table(name string, args ...any) IRepository          { return d }
func (d *DummyRepo) Count(count *int64) error {
	*count = 0
	return nil
}

func dummyRepository(db *gorm.DB) IRepository {
	return &DummyRepo{}
}

// =======================
// Tests for Gorm
// =======================

// newTestDatabaseContext creates a valid DatabaseContext using in-memory SQLite.
func newTestDatabaseContext() DatabaseContext {
	dbCtx, err := NewDatabaseContext(":memory:", "sqlite", "info")
	if err != nil {
		panic(fmt.Sprintf("failed to create DatabaseContext: %v", err))
	}
	return *dbCtx
}

// TestNewGormSuccess verifies successful initialization of NewGorm.
func TestNewGormSuccess(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "sqlquery_*.sql")
	assert.NoError(t, err, "Failed to create temporary file")
	defer os.Remove(tmpFile.Name())

	sqlContent := "SELECT 1;"
	_, err = tmpFile.WriteString(sqlContent)
	assert.NoError(t, err, "Failed to write to temporary file")
	tmpFile.Close()

	sqlQueryPaths := map[string]string{"dummy": tmpFile.Name()}
	seedQueries := []string{}
	dbCtx := newTestDatabaseContext()

	g, err := NewGorm(dbCtx, dummyRepository, seedQueries, sqlQueryPaths)
	assert.NoError(t, err, "Unexpected error from NewGorm")

	query, err := g.GetQuery("dummy")
	assert.NoError(t, err, "Failed to retrieve query")
	assert.Equal(t, sqlContent, query, "Query content mismatch")

	db := g.GetDB()
	_, ok := db.(*DummyRepo)
	assert.True(t, ok, "Expected DummyRepo, got %T", db)
}

// TestNewGormDialectorFailure verifies error when GetDialector fails.
func TestNewGormDialectorFailure(t *testing.T) {
	dbCtx := DatabaseContext{
		dsn:         "dummy",
		driver:      999,
		loggerLevel: "info",
	}

	_, err := NewGorm(dbCtx, dummyRepository, []string{}, map[string]string{})
	assert.Error(t, err, "Expected error due to dialector failure")
	assert.Contains(t, err.Error(), "failed to get dialector")
}

// TestCacheSQLQueriesFailure verifies failure when reading a non-existent SQL file.
func TestCacheSQLQueriesFailure(t *testing.T) {
	sqlQueryPaths := map[string]string{"nonexistent": "/nonexistent/path.sql"}
	dbCtx := newTestDatabaseContext()

	_, err := NewGorm(dbCtx, dummyRepository, []string{}, sqlQueryPaths)
	assert.Error(t, err, "Expected error due to SQL file read failure")
	assert.Contains(t, err.Error(), "failed to cache SQL queries")
}

// TestSeedFailure verifies error when executing seed with a non-existent file.
func TestSeedFailure(t *testing.T) {
	seedQueries := []string{"/nonexistent/path_seed.sql"}
	sqlQueryPaths := map[string]string{}
	dbCtx := newTestDatabaseContext()

	g, err := NewGorm(dbCtx, dummyRepository, seedQueries, sqlQueryPaths)
	assert.NoError(t, err, "Unexpected error from NewGorm")

	err = g.Seed()
	assert.Error(t, err, "Expected error when executing Seed with non-existent file")
}

// TestMigrateSuccess verifies the migration process.
func TestMigrateSuccess(t *testing.T) {
	sqlQueryPaths := map[string]string{}
	seedQueries := []string{}
	dbCtx := newTestDatabaseContext()

	g, err := NewGorm(dbCtx, dummyRepository, seedQueries, sqlQueryPaths)
	assert.NoError(t, err, "Unexpected error from NewGorm")

	type dummyModel struct {
		ID int
	}

	err = g.Migrate(&dummyModel{})
	assert.NoError(t, err, "Migration failed")

	var tableName string
	err = g.connection.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name='dummy_models';").Scan(&tableName).Error
	assert.NoError(t, err, "Failed to query sqlite_master")
	assert.NotEmpty(t, tableName, "Table for DummyModel was not created")
}
