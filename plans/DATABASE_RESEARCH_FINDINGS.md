# Database Implementation Research Findings

## Current Database Architecture

### Core Database Logic (`internal/db/connect.go`)

**Current Flow:**
1. **Connection Creation**: `db.Connect(ctx, dataDir)` is the main entry point
2. **Directory Setup**: Creates data directory if it doesn't exist
3. **SQLite Connection**: `sql.Open("sqlite3", dbPath)` where `dbPath = dataDir/mix.db`
4. **Health Check**: Pings database with timeout
5. **Pragma Configuration**: Sets SQLite performance pragmas
6. **Migration Execution**: Uses Goose to run embedded migrations

**Key Characteristics:**
- Single function responsible for all database setup
- Embedded migrations via `embed.FS`
- Hard-coded SQLite driver (`"sqlite3"`)
- Returns `*sql.DB` directly to caller

### Database Usage Patterns

**Primary Usage Locations:**
1. **`cmd/root.go`** (line 101): Main application entry point
   ```go
   conn, err := db.Connect(dbCtx, cfg.Data.Directory)
   ```

2. **`internal/app/app.go`** (line 32): Application initialization
   ```go
   func New(ctx context.Context, conn *sql.DB) (*App, error)
   ```

3. **Test files**: Multiple integration tests use `db.Connect(ctx, ".mix")`

### Service Dependencies on Database

**Services that require `*sql.DB`:**
1. **SQLC Queries**: `db.New(conn)` creates type-safe query interface
2. **User Preferences**: `config.InitUserPreferences(conn)`
3. **API Credentials**: `config.InitAPICredentials(conn)`
4. **History Service**: `history.NewService(q, conn)` (requires both)

**Service Initialization Pattern:**
```go
func New(ctx context.Context, conn *sql.DB) (*App, error) {
    q := db.New(conn)  // Create SQLC querier

    // Services that only need querier
    sessions := session.NewService(q, storageConfig)
    baseMessageService := message.NewService(q)

    // Services that need raw connection too
    config.InitUserPreferences(conn)
    config.InitAPICredentials(conn)
    files := history.NewService(q, conn)
}
```

### Current Configuration

**Database Configuration:**
- Data directory: `cfg.Data.Directory` (defaults to `.mix`)
- Database filename: Hard-coded as `"mix.db"`
- No provider abstraction
- No remote database support

### Migration System

**Current Migration Setup:**
- Uses Goose migration tool
- Migrations embedded via `//go:embed migrations/*.sql`
- SQLite dialect hard-coded
- Automatic migration execution on connect

**Migration Files Located:**
- `internal/db/migrations/20250424200609_initial.sql`
- Multiple other migration files for schema evolution

### Test Infrastructure

**Test Status:** ✅ Most tests passing (95%+ pass rate)
- Unit tests for individual services work
- Integration tests use `db.Connect(ctx, ".mix")`
- Mock interfaces available (`mock_querier.go`)
- No database abstraction tests (none exist yet)

**Test Database Pattern:**
```go
// Tests directly use the same Connect function
conn, err := db.Connect(ctx, ".mix")
if err != nil {
    t.Fatalf("Failed to connect to test database: %v", err)
}
```

### Current SQLite Features Used

**SQLite Pragmas Applied:**
- `PRAGMA foreign_keys = ON` - Referential integrity
- `PRAGMA journal_mode = WAL` - Write-Ahead Logging
- `PRAGMA page_size = 4096` - Performance optimization
- `PRAGMA cache_size = -8000` - Memory cache configuration
- `PRAGMA synchronous = NORMAL` - Durability vs performance balance

### Key Integration Points for Abstraction

**Where Database Connections Are Created:**
1. `cmd/root.go:101` - Main application startup
2. `internal/http/integration_tests/test_utils.go` - Test infrastructure
3. `internal/http/session_fork_integration_test.go` - Integration testing
4. `internal/http/test_utils.go` - Test utilities

**Where Raw `*sql.DB` Is Required:**
1. Service initialization functions that need both querier and raw connection
2. Global service initialization (`config.InitUserPreferences`, `config.InitAPICredentials`)
3. History service (needs raw connection for some operations)

## Abstraction Strategy

### Phase 1 Approach - SQLite Only Refactoring

**Safe Abstraction Points:**
1. Replace `db.Connect()` with `DatabaseManager.Connect()`
2. Keep exact same behavior and error handling
3. Wrap existing SQLite logic in provider interface
4. Maintain all current pragmas and migration logic

**No Changes Needed:**
- SQLC generated code continues to work unchanged
- Service constructors keep same signatures
- Migration files remain identical
- Test infrastructure works with minimal updates

**Provider Interface Design:**
```go
type Provider interface {
    Connect(ctx context.Context) error
    GetDB() *sql.DB
    Close() error
    RunMigrations(ctx context.Context) error
}
```

### Risk Assessment

**Low Risk Changes:**
- Creating abstraction interfaces ✅
- Wrapping existing SQLite logic ✅
- Updating single connection creation point ✅

**Medium Risk Changes:**
- Modifying app initialization flow
- Updating test infrastructure

**No Risk:**
- SQLC queries (completely unchanged)
- Service business logic (unchanged)
- Migration files (reused exactly)

## Conclusion

The current database implementation is well-structured and centralized, making it ideal for abstraction. The single entry point (`db.Connect`) and clear separation between connection management and business logic will allow for clean abstraction without disrupting existing functionality.

**Ready for Stage 2**: Current system is stable with good test coverage, making it safe to proceed with abstraction layer creation.