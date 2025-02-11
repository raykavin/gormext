package pkg

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"
)

const SQLFileExtension = ".sql"

// IRepository defines an interface for repository operations.
type IRepository interface {
	WithTransaction(fn func(tx IRepository) error) error // Execute operations within a transaction.
	WithContext(ctx context.Context) IRepository         // Set context for queries.
	FirstByID(id any, dest any) error                    // Find a record by its ID.
	First(dest any, conds ...any) error                  // Return the first record that matches the condition.
	Find(dest any) error                                 // Find all records.
	Create(entity any) error                             // Create a new record.
	Update(entity any) error                             // Update an existing record.
	Delete(entity any) error                             // Delete a record.
	Exec(sql string, value ...any) error                 // Execute a SQL query.
	IDEqual(id any) IRepository                          // Add condition "ID = ?".
	IDIn(ids []any) IRepository                          // Add condition "ID IN (?)".
	Where(query any, args ...any) IRepository            // Add a WHERE clause.
	Joins(query string, args ...any) IRepository         // Add a JOIN clause.
	Preload(query string, args ...any) IRepository       // Add a PRELOAD clause.
	Order(value any) IRepository                         // Add an ORDER BY clause.
	IsActive() IRepository                               // Filter records where "active IS TRUE".
	Table(name string, args ...any) IRepository          // Specify the table to query.
	Count(count *int64) error                            // Count records matching the query.
}

// Repository is a function type that receives a *gorm.DB connection and returns an IRepository.
type Repository func(*gorm.DB) IRepository

// Config wraps the GORM configuration.
type Config struct {
	gorm.Config
}

// Gorm encapsulates the database connection and additional functionalities.
type Gorm struct {
	connection  *gorm.DB
	sqlQueries  *sync.Map
	databaseCtx DatabaseContext
	repository  Repository
	seedQueries []string
}

// NewGorm initializes a new instance of Gorm.
func NewGorm(
	databaseCtx DatabaseContext,
	repository Repository,
	seedQueryPaths []string,
	sqlQueryPaths map[string]string,
	config ...Config,
) (*Gorm, error) {
	dialector, err := databaseCtx.GetDialector()
	if err != nil {
		return nil, fmt.Errorf("failed to get dialector: %w", err)
	}

	gormConfig := &gorm.Config{}
	if len(config) > 0 {
		gormConfig = &config[0].Config
	}

	conn, err := gorm.Open(dialector(), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	g := &Gorm{
		connection:  conn,
		databaseCtx: databaseCtx,
		repository:  repository,
		seedQueries: seedQueryPaths,
		sqlQueries:  &sync.Map{},
	}

	if err := g.cacheSQLQueries(sqlQueryPaths); err != nil {
		return nil, fmt.Errorf("failed to cache SQL queries: %w", err)
	}

	return g, nil
}

// Seed executes seed queries to initialize the database.
func (g *Gorm) Seed() error {
	for _, queryPath := range g.seedQueries {
		content, err := os.ReadFile(queryPath)
		if err != nil {
			return fmt.Errorf("failed to read seed file '%s': %w", queryPath, err)
		}

		if err := g.connection.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("failed to execute seed query from file '%s': %w", queryPath, err)
		}

		time.Sleep(10 * time.Millisecond) // Throttle to avoid overwhelming the database.
	}
	return nil
}

// GetQuery retrieves a cached SQL query by name.
func (g *Gorm) GetQuery(name string) (string, error) {
	cachedQuery, found := g.sqlQueries.Load(name)
	if !found {
		return "", fmt.Errorf("sql query '%s' not found", name)
	}

	queryStr, ok := cachedQuery.(string)
	if !ok {
		return "", fmt.Errorf("invalid type for sql query '%s'", name)
	}

	return queryStr, nil
}

// GetDB returns a repository instance for database operations.
func (g *Gorm) GetDB() IRepository {
	return g.repository(g.connection)
}

// Migrate runs auto-migration for the given models.
func (g *Gorm) Migrate(models ...any) error {
	return g.connection.AutoMigrate(models...)
}

// cacheSQLQueries reads and stores SQL queries based on the provided file paths.
func (g *Gorm) cacheSQLQueries(queriesPaths map[string]string) error {
	for name, path := range queriesPaths {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read SQL file '%s': %w", path, err)
		}

		g.sqlQueries.Store(name, string(content))
	}
	return nil
}
