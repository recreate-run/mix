# Database Abstraction Layer for Turso libsql Support

## Overview

This plan outlines the implementation of a database abstraction layer to support both local SQLite and remote Turso libsql backends in the Mix agent. The design maintains backward compatibility while enabling cloud-native deployment scenarios with Turso as the remote database provider.

## Current Architecture Analysis

### Database Layer
- **Current DB**: SQLite using `ncruces/go-sqlite3` driver
- **Query Generation**: SQLC for type-safe SQL queries
- **Migrations**: Goose migration system
- **Storage Location**: Local file at `{dataDir}/mix.db`

### Recent Refactoring Benefits
The recent refactoring introduced several patterns that align well with our abstraction layer:

1. **Interface-based Services**: `preferences.Service` interface pattern supports dependency injection
2. **Centralized Initialization**: Database connection handled in `app.New()`
3. **Service-Oriented Architecture**: Each domain has its own service with clear boundaries
4. **Configuration Management**: Centralized through config package

### Data Models
- **Sessions**: Core conversation units with metadata
- **Messages**: Chat messages linked to sessions
- **Files**: File metadata and content tracking
- **API Credentials**: Encrypted provider API keys
- **OAuth Credentials**: OAuth tokens and refresh tokens
- **User Preferences**: Agent configurations and settings

## Implementation Plan

### Phase 1: Database Abstraction Interface

Create a unified database provider interface that abstracts connection management:

```go
// internal/database/provider.go
package database

import (
    "context"
    "database/sql"
)

type Provider interface {
    // Connection management
    Connect(ctx context.Context) error
    Close() error
    GetDB() *sql.DB
    Ping(ctx context.Context) error

    // Transaction support
    BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)

    // Provider identification
    Type() DatabaseType

    // Migration support
    RunMigrations(ctx context.Context) error
}

type DatabaseType string

const (
    DatabaseLocal  DatabaseType = "local"  // SQLite
    DatabaseRemote DatabaseType = "remote" // Turso
)
```

### Phase 2: Configuration Enhancement

Extend existing configuration to support database backends:

```go
// internal/config/database.go
type DatabaseConfig struct {
    Type   DatabaseType `json:"type"`   // "local" or "remote"
    Local  LocalConfig  `json:"local"`  // SQLite configuration
    Remote TursoConfig  `json:"remote"` // Turso configuration
}

type LocalConfig struct {
    DataDir  string `json:"dataDir"`
    Filename string `json:"filename"` // defaults to "mix.db"
}

type TursoConfig struct {
    URL       string `json:"url"`       // libsql://[databaseId]-[organizationName].turso.io
    AuthToken string `json:"authToken"` // Turso auth token
}
```

### Phase 3: Provider Implementations

#### SQLite Provider
```go
// internal/database/sqlite_provider.go
type SQLiteProvider struct {
    config LocalConfig
    db     *sql.DB
}

func NewSQLiteProvider(config LocalConfig) *SQLiteProvider {
    return &SQLiteProvider{config: config}
}

func (p *SQLiteProvider) Connect(ctx context.Context) error {
    // Current SQLite connection logic from internal/db/connect.go
    dbPath := filepath.Join(p.config.DataDir, p.config.Filename)
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return fmt.Errorf("failed to open SQLite database: %w", err)
    }
    p.db = db
    return nil
}

func (p *SQLiteProvider) Type() DatabaseType {
    return DatabaseLocal
}
```

#### Turso Provider
```go
// internal/database/turso_provider.go
type TursoProvider struct {
    config TursoConfig
    db     *sql.DB
}

func NewTursoProvider(config TursoConfig) *TursoProvider {
    return &TursoProvider{config: config}
}

func (p *TursoProvider) Connect(ctx context.Context) error {
    // Connect to Turso using libsql driver
    db, err := sql.Open("libsql", p.config.URL+"?authToken="+p.config.AuthToken)
    if err != nil {
        return fmt.Errorf("failed to connect to Turso: %w", err)
    }
    p.db = db
    return nil
}

func (p *TursoProvider) Type() DatabaseType {
    return DatabaseRemote
}
```

### Phase 4: Database Manager

Implement simple database management:

```go
// internal/database/manager.go
type Manager struct {
    provider Provider
    config   DatabaseConfig
}

func NewManager(config DatabaseConfig) (*Manager, error) {
    var provider Provider

    switch config.Type {
    case DatabaseLocal:
        provider = NewSQLiteProvider(config.Local)
    case DatabaseRemote:
        provider = NewTursoProvider(config.Remote)
    default:
        return nil, fmt.Errorf("unsupported database type: %s", config.Type)
    }

    return &Manager{
        provider: provider,
        config:   config,
    }, nil
}

func (m *Manager) Connect(ctx context.Context) error {
    return m.provider.Connect(ctx)
}

func (m *Manager) GetDB() *sql.DB {
    return m.provider.GetDB()
}

func (m *Manager) Close() error {
    return m.provider.Close()
}

func (m *Manager) RunMigrations(ctx context.Context) error {
    return m.provider.RunMigrations(ctx)
}
```

### Phase 5: Migration System Enhancement

Extend Goose migrations to work with both backends:

```go
// Update existing internal/db/connect.go
func (p *SQLiteProvider) RunMigrations(ctx context.Context) error {
    goose.SetBaseFS(FS)

    if err := goose.SetDialect("sqlite3"); err != nil {
        return fmt.Errorf("failed to set dialect: %w", err)
    }

    migrationCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    return goose.UpContext(migrationCtx, p.db, "migrations")
}

func (p *TursoProvider) RunMigrations(ctx context.Context) error {
    goose.SetBaseFS(FS)

    // Turso uses SQLite dialect
    if err := goose.SetDialect("sqlite3"); err != nil {
        return fmt.Errorf("failed to set dialect: %w", err)
    }

    migrationCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    return goose.UpContext(migrationCtx, p.db, "migrations")
}
```

### Phase 6: Integration with Existing Architecture

Update the application initialization to use the new abstraction:

```go
// cmd/root.go modifications
func rootCmd.RunE(cmd *cobra.Command, args []string) error {
    // Load configuration with database settings
    cfg, err := config.Load(cwd, debug, skipPermissions)
    if err != nil {
        return err
    }

    // Create database manager
    dbManager, err := database.NewManager(cfg.Database)
    if err != nil {
        return fmt.Errorf("failed to create database manager: %w", err)
    }

    // Connect with timeout
    dbCtx, dbCancel := context.WithTimeout(ctx, 30*time.Second)
    defer dbCancel()

    if err := dbManager.Connect(dbCtx); err != nil {
        return fmt.Errorf("failed to connect to database: %w", err)
    }
    defer dbManager.Close()

    // Run migrations
    if err := dbManager.RunMigrations(dbCtx); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    // Create app with database connection
    app, err := app.New(ctx, dbManager.GetDB())
    if err != nil {
        return err
    }
    defer app.Shutdown()

    // Rest remains the same...
}
```

## Configuration Integration

### Environment Variables
```bash
# Local SQLite (default)
MIX_DB_TYPE=local
MIX_DB_LOCAL_DATA_DIR=.mix
MIX_DB_LOCAL_FILENAME=mix.db

# Remote Turso
MIX_DB_TYPE=remote
MIX_DB_TURSO_URL=libsql://my-database-org.turso.io
MIX_DB_TURSO_AUTH_TOKEN=eyJ...
```

### Configuration Loading
```go
// internal/config/config.go additions
type Config struct {
    // Existing fields...
    Database DatabaseConfig `json:"database"`
}

func Load(sessionStorageDir string, debug bool, skipPermissions bool) (*Config, error) {
    // Existing logic...

    cfg.Database = loadDatabaseConfig()
    return cfg, nil
}

func loadDatabaseConfig() DatabaseConfig {
    dbType := DatabaseType(getEnvOrDefault("MIX_DB_TYPE", string(DatabaseLocal)))

    config := DatabaseConfig{
        Type: dbType,
        Local: LocalConfig{
            DataDir:  getEnvOrDefault("MIX_DB_LOCAL_DATA_DIR", ".mix"),
            Filename: getEnvOrDefault("MIX_DB_LOCAL_FILENAME", "mix.db"),
        },
        Remote: TursoConfig{
            URL:       os.Getenv("MIX_DB_TURSO_URL"),
            AuthToken: os.Getenv("MIX_DB_TURSO_AUTH_TOKEN"),
        },
    }

    return config
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

## Implementation Benefits

### 1. Backward Compatibility
- Existing SQLite deployments continue working unchanged
- No breaking changes to existing APIs or data models
- Configuration defaults to local SQLite mode

### 2. Cloud-Native Support
- Enables distributed deployment with shared Turso database
- Supports multi-instance horizontally scaled deployments
- Provides consistent data access across instances

### 3. Simplicity
- Clean abstraction with minimal complexity
- Easy switching between local and remote modes
- No hybrid mode complexity to maintain

### 4. Developer Experience
- Transparent to existing service implementations
- No changes required to SQLC queries or business logic
- Simple configuration management

### 5. Operational Benefits
- Centralized database management for cloud deployments with Turso
- Built-in backup and disaster recovery through Turso
- Global edge deployment capabilities

## Implementation Strategy - Staged Approach

### Stage 1: Research & Analysis (Days 1-2)
**Objective**: Thoroughly understand current database implementation
- Deep dive into existing database connection logic (`internal/db/connect.go`)
- Analyze all database-related initialization in `app.New()` and `cmd/root.go`
- Document current SQLite usage patterns and dependencies
- Map out all places where database connections are created/used
- Identify potential integration points for abstraction layer

### Stage 2: Current System Validation (Day 2)
**Objective**: Ensure existing system is stable before changes
- Run complete test suite and verify all tests pass
- Document any existing test failures or issues
- Validate current SQLite behavior matches expectations
- Create baseline performance measurements

### Stage 3: Abstraction Layer - SQLite Only (Days 3-4)
**Objective**: Create abstraction without changing behavior
- Design and implement database provider interface
- Create SQLite provider wrapping existing logic exactly
- Implement simple database manager (SQLite only)
- NO new features, just clean abstraction of current code

### Stage 4: Refactoring Integration (Days 4-5)
**Objective**: Replace direct database calls with abstraction
- Update `cmd/root.go` to use database manager
- Ensure `app.New()` works with abstracted connection
- Replace direct `db.Connect()` calls with manager
- Maintain exact same behavior and error handling

### Stage 5: Validation & Testing (Day 5)
**Objective**: Prove abstraction layer works perfectly
- Run complete test suite - all tests must still pass
- Verify application behavior is identical to before
- Performance testing to ensure no regressions
- Fix any issues before proceeding

### Stage 6: Turso Provider Addition (Days 6-7)
**Objective**: Add Turso support to proven abstraction
- Implement Turso provider using libsql client
- Extend database manager to support provider selection
- Add configuration support for Turso connection strings
- Update migration system for Turso compatibility

### Stage 7: Final Integration & Testing (Days 7-8)
**Objective**: Complete integration with comprehensive testing
- End-to-end testing with both SQLite and Turso
- Configuration validation and error handling
- Performance comparison between providers
- Documentation and deployment guides

## Dependencies

### New Dependencies
```go
// go.mod additions
require (
    github.com/tursodatabase/libsql-client-go v0.0.0-latest
)
```

### Existing Dependencies (maintained)
- `github.com/ncruces/go-sqlite3` for local SQLite
- `github.com/pressly/goose/v3` for migrations
- All existing SQLC generated code remains unchanged

## Testing Strategy

### Unit Tests
- Provider interface implementations
- Database manager logic
- Configuration loading and validation
- Migration system for both backends

### Integration Tests
- End-to-end tests with both SQLite and Turso backends
- Migration testing across both database types
- Performance and load testing
- Connection reliability testing

### Compatibility Tests
- Existing test suite should pass unchanged
- Data migration tests between backends
- Schema validation across providers

This plan provides a clean, simplified foundation for adding Turso libsql support to the Mix agent while leveraging the recent refactoring improvements and maintaining the existing clean architecture patterns. The removal of hybrid mode complexity makes the implementation more maintainable and easier to reason about.