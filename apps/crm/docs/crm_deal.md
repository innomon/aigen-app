# CRM Deal Context

A `crm_deal` represents a qualified opportunity or potential sale currently active in the sales pipeline.

## Lifecycle
Deals are created after a lead is qualified. They progress through various stages defined by the `status` field (linked to `crm_deal_status`). The goal is to move the deal towards a `closed_date`.
Key metrics track the `expected_closure_date`, `expected_deal_value`, and a success `probability` percentage.

## Relationships
- **Organization**: Must be linked to a `crm_organization` representing the customer.
- **Status**: Must map to a valid `crm_deal_status`.
- **Industry**: Maps to a valid `crm_industry` for reporting purposes.
- **Products**: Can include multiple line items linked via the `crm_products` collection.

## Calculations & Financials
- `deal_value` and `expected_deal_value` represent the monetary worth of the opportunity.
- The `total` and `net_total` fields are calculated based on the associated products.