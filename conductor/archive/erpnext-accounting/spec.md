# Specification: ERPNext Accounting Transformation

## Objective
Translate the core ERPNext accounting DocTypes into the `AiGen CMS` framework's `Entity` and `Schema` system.

## Scope
The transformation includes the following core DocTypes:
1. `Account` (Chart of Accounts)
2. `GL Entry` (General Ledger Entries)
3. `Journal Entry` (including child table `Journal Entry Account`)
4. `Fiscal Year`
5. `Cost Center`
6. `Company` (Required for accounting context)
7. `Currency`

## Mapping Rules

### DocType to Entity
- `name` in DocType maps to `Name` and `DisplayName` in `Entity`.
- `table` in DocType maps to `TableName` in `Entity`.
- `fields` in DocType map to `Attributes` in `Entity`.

### Field Type to DataType
- `Data`, `Long Text`, `Small Text`, `Text` -> `String`
- `Int`, `Check` -> `Int` (or `Boolean` for Check)
- `Float`, `Currency` -> `Float` (Need to verify if `AiGen CMS` supports `Float`)
- `Date`, `Datetime` -> `Datetime`
- `Link` -> `Lookup`
- `Table` -> `Collection`

### Child Tables
- DocTypes with `fieldtype: Table` will be represented as a `Collection` attribute pointing to a child entity.

## Implementation Details
- Define the `Entity` descriptors for each DocType.
- Create JSON representations for the `Schema` settings.
- Implement Go services for handling business logic (if needed, though initial phase is data structure translation).
