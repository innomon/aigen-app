# CRM Products Context

`crm_products` represents a specific product line item associated with a `crm_lead` or `crm_deal`.

## Lifecycle
These items are typically created when a sales rep adds a product from the catalog (`crm_product`) to a specific deal or lead. 

## Key Fields & Calculations
- **Product Code**: Lookup to the `crm_product` master catalog.
- **Quantity & Rate**: Define the base pricing for this line item.
- **Discount (%)**: A percentage-based reduction applied to the line item.
- **Amount & Net Amount**: Calculated fields representing the subtotal and total after discounts for this line.

## Relationships
- **Parent ID / Parent Type**: These fields maintain the polymorphic relationship, allowing the same products table to serve both `crm_lead` and `crm_deal` parents.