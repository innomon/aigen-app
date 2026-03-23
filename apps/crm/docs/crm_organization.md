# CRM Organization Context

A `crm_organization` represents a qualified business entity or customer. It is typically created when a `crm_lead` is successfully converted.

## Lifecycle
Organizations are long-lived entities within the CRM. They act as the central hub for associated contacts, deals, and communication history.

## Relationships
- **Industry**: Maps to a valid `crm_industry`.
- **Deals**: Can have multiple associated `crm_deal` entities linking back to this organization.

## Key Fields
- Contains corporate details such as `organization_name`, `website`, `annual_revenue`, and `no_of_employees`.
- Tracks financial metadata like default `currency` and `exchange_rate` for cross-border deals.