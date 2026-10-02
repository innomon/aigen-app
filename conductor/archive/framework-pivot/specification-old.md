# FormCMS Go Rewrite Specification

## 1. Problem and Scope
The goal of FormCMS is to build a headless CMS framework that avoids rebuilding common features and simplifies development workflow. This project will rewrite the original ASP.NET Core backend into Go for better performance, smaller memory footprint, and deployment simplicity while retaining all core features.

## 2. Core Features
1. **Dynamic Schema & Data Modeling**: Admins can define entities, attributes, and relationships (1:N, N:1, N:N) which are mapped to physical database tables to leverage native indexing and performance.
2. **RESTful APIs**: Automatically expose CRUD APIs for dynamically generated entities.
3. **GraphQL Queries**: Expose optimized GraphQL APIs for frontend consumption, solving N+1 query problems. Convert stored GraphQL queries to REST endpoints for CDN caching.
4. **Drag and Drop Page Designer**: Integrate GrapesJS to visually build pages, bound to backend data queries via Handlebars templates.
5. **Assets Library**: Handle file/image uploads, large file chunking, and extensible storage providers (Local/Cloud). Include automatic image compression.
6. **Social Engagement**: Out-of-the-box features for likes, views, shares, saves, comments, notifications, and popularity score calculation. Use buffered writes (In-Memory or Redis) to scale to thousands of QPS.

## 3. Architecture
- **Language**: Go 1.22+
- **Web Framework**: `go-chi/chi` (Standard library compatible routing)
- **Database Abstraction**: `database/sql` coupled with `Masterminds/squirrel` for dynamic SQL building (equivalent to SqlKata).
- **GraphQL Engine**: `graphql-go/graphql`
- **Template Engine**: `aymerick/raymond` (for compiling Handlebars in Go)
- **Supported Databases**: PostgreSQL, MySQL, SQLite, SQL Server
- **Image Processing**: `bimg` or `disintegration/imaging`
- **Frontend**: Embed the existing React Admin Panel (`FormCmsAdminApp`) and jQuery Schema Builder inside the Go binary using `embed`.

## 4. Key Differences from C# Implementation
- **Dependency Injection**: Replaced by Go's explicit struct initialization and interface composition.
- **Background Workers**: Utilize Go routines and channels instead of `IHostedService`.
- **Entity Framework**: Will not be used. Pure dynamic SQL mapping via `squirrel` to handle dynamic DDL and DML operations.