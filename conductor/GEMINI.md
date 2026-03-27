# AIGenApp Context

## Mission
You are evolving `AIGenApp` (formerly `aigen-cms` / `FormCMS`) backend in Go (Golang). The project maintains a headless CMS and dynamic application framework with schema-on-read data modeling, GraphQL support, a page designer (GrapesJS), and extensive user engagement tracking (views, likes, comments).

## Important Architectural Decisions
- **Framework**: `net/http` + `chi` for routing.
- **Data Model**: All entities are stored in a single table (`aigen_records`) utilizing a JSON schema structure (Namespace, Key, Rec, MetaData). 
- **SQL Building**: Use `Masterminds/squirrel` for queries inside SQL-based DAOs (Postgres, SQLite). Services MUST NOT use squirrel or direct SQL; they must rely exclusively on `IPrimaryDao` methods.
- **GraphQL**: Use `graphql-go/graphql`.
- **Database**: The abstraction layer (`IPrimaryDao`) supports PostgreSQL, SQLite, and Google Cloud Firestore natively utilizing their JSON/document capabilities.
- **Template Engine**: `aymerick/raymond` for Handlebars templates.

## Important Rules
- Favor simple, clean Go idioms over overly complex abstractions.
- We use a single-table JSON architecture rather than creating physical tables at runtime. Do NOT write code that executes `CREATE TABLE` or `ALTER TABLE` dynamically for user schemas.
- Ensure secure JSON path construction and query building to prevent injection. Use parameterized values with squirrel.
- Concurrency and background workers should be handled using standard goroutines and channels, rather than heavy background worker frameworks unless necessary.
- Store static assets and embedded files (like the admin panel frontend) using Go `//go:embed`.

## Workflow
1. Use `codebase_investigator` to search source for business logic.
2. Ensure new features adhere to the `RecJSON` single-table persistence pattern.
3. Keep test cases robust using `testing` package and `testify`.