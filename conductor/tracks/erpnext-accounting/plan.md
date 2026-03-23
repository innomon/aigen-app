# Implementation Plan: ERPNext Accounting Transformation

## Phase 1: Research & Mapping (Done)
- [x] Analyze ERPNext accounting DocTypes.
- [x] Analyze `AiGen CMS` Entity/Schema system.
- [x] Define mapping rules.

## Phase 2: Schema Definition
- [x] Create `Entity` descriptors in Go for:
    - [x] Company
    - [x] Currency
    - [x] Account
    - [x] GL Entry
    - [x] Journal Entry
    - [x] Journal Entry Account (Child Table)
    - [x] Fiscal Year
    - [x] Cost Center
- [x] Create JSON Schema files for these entities. (Handled dynamically via setup.go)

## Phase 3: Framework Enhancements
- [x] Add `Float` and `Decimal` support to `AiGen CMS`.
    - [x] Update `descriptors.DataType`.
    - [x] Update `datamodels.ColumnType`.
    - [x] Update `sqlite.go` and `postgres.go` to support FLOAT/DECIMAL.
    - [x] Update `displaymodels.DisplayType` if needed.

## Phase 4: Implementation
- [x] Implement the `Entity` registration in `AiGen CMS`.
- [x] Verify schema creation in the database.

## Phase 5: Validation
- [ ] Create test data for each entity.
- [ ] Verify that basic CRUD operations work for the translated accounting entities.
