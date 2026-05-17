# AIGenApp Context

## Mission
You are evolving `AIGenApp` (formerly `aigen-cms` / `FormCMS`) backend in Go (Golang). The project maintains a headless CMS and dynamic application framework with schema-on-read data modeling, GraphQL support, a page designer (GrapesJS), and extensive user engagement tracking (views, likes, comments).

## Important Architectural Decisions
- **Framework**: `net/http` + `chi` for routing.
- **Data Model**: All entities are stored in a single table (`aigen_records`) utilizing a JSON schema structure (Namespace, Key, Rec, MetaData). 
- **SQL Building**: Use `Masterminds/squirrel` for queries inside SQL-based DAOs (Postgres). Services MUST NOT use squirrel or direct SQL; they must rely exclusively on `IPrimaryDao` methods.
- **GraphQL**: Use `graphql-go/graphql`.
- **Database**: The abstraction layer (`IPrimaryDao`) supports PostgreSQL and Google Cloud Firestore natively utilizing their JSON/document capabilities.
- **Template Engine**: `aymerick/raymond` for Handlebars templates.

### Go Development
- **Style**: Adhere strictly to [Effective Go](https://go.dev/doc/effective_go) and Go Code Review Comments.
- **Formatting**: Always use `gofmt` and `goimports`.
- **Error Handling**: Never ignore errors with `_`. Handle them explicitly and return early to reduce nesting.
- **Testing**: Use table-driven tests with `t.Run()`. Tests must reside in the same package as the code they test.
- **Concurrency**: Use `context.Context` as the first parameter for functions involving cancellation or timeouts.
- use on postgres database, **DO NOT** use sqilite for this project.
- never use the spf13 library (Cobra/Pflag). Instead, always implement a handcrafted command registry for CLI and slash commands.


## Important Rules
- Favor simple, clean Go idioms over overly complex abstractions.
- We use a single-table JSON architecture rather than creating physical tables at runtime. Do NOT write code that executes `CREATE TABLE` or `ALTER TABLE` dynamically for user schemas.
- Ensure secure JSON path construction and query building to prevent injection. Use parameterized values with squirrel.
- Concurrency and background workers should be handled using standard goroutines and channels, rather than heavy background worker frameworks unless necessary.
- Store static assets and embedded files (like the admin panel frontend) using Go `//go:embed`.

## Downstream App Development
When tasked with creating a new downstream app (e.g., in `apps/`), refer to the [Downstream App Development Guide](conductor/downstream-app-development-guide.md) for step-by-step instructions on manifests, schemas, and test data.

## Workflow
1. Use `codebase_investigator` to search source for business logic.
2. Ensure new features adhere to the `RecJSON` single-table persistence pattern.
3. Keep test cases robust using `testing` package and `testify`.