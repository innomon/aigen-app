# AiGen CMS Rewrite Context

## Mission
You are migrating the `AiGen CMS` (formerly `FormCMS`) backend from C# (ASP.NET Core) to Go (Golang). The project maintains a headless CMS with dynamic data modeling (database tables created dynamically based on user schema), GraphQL support, a page designer (GrapesJS), and extensive user engagement tracking (views, likes, comments).

## Important Architectural Decisions
- **Framework**: `net/http` + `chi` for routing.
- **SQL Building**: Use `Masterminds/squirrel` for dynamic queries since table schema changes at runtime.
- **GraphQL**: Use `graphql-go/graphql`.
- **Database**: Use standard `database/sql` driver mechanism to support SQLite, PostgreSQL, MySQL, and SQL Server.
- **Template Engine**: `aymerick/raymond` for Handlebars templates to match the original C# `Handlebars.Net`.

## Important Rules
- Favor simple, clean Go idioms over overly complex abstractions.
- Avoid ORMs (like GORM) for core dynamic entity operations because table definitions aren't known at compile-time. Use raw queries and query builders like `squirrel`.
- Ensure secure SQL construction to prevent SQL injection when creating physical tables from user-provided schema names.
- Concurrency and background workers should be handled using standard goroutines and channels, rather than heavy background worker frameworks unless necessary.
- Store static assets and embedded files (like the admin panel frontend) using Go `//go:embed`.

## Workflow
1. Use `codebase_investigator` to search original C# source for business logic.
2. Implement corresponding logic in Go.
3. Keep test cases similar to original C# test cases but written in `testing` package (and potentially `testify`).