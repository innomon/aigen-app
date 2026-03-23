# CRM Lead Context

A `crm_lead` represents an unqualified prospect in the CRM pipeline.

## Lifecycle
Leads are typically generated from marketing campaigns, contact forms, or manual entry. Their progress is tracked using the `status` field, which looks up to the `crm_lead_status` entity.
A lead is evaluated based on `industry`, `no_of_employees`, and `annual_revenue`. Once a lead is successfully qualified, the `converted` flag should be set to true, and it should typically be converted into a `crm_organization` and/or a `crm_deal`.

## Relationships
- **Status**: Must map to a valid `crm_lead_status`.
- **Industry**: Must map to a valid `crm_industry`.
- **Products**: Can have multiple interested products linked via the `crm_products` collection.

## Constraints
- Often requires basic contact information such as `email`, `mobile_no`, or `phone`.
- The `lead_name` is typically constructed or provided as the primary identifier.