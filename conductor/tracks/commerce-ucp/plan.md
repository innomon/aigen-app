# Implementation Plan: Commerce UCP & AP2 Integration

## Phase 1: BizDef Definition
- [ ] Define `commerce_ucp` entities in `bizdefs/commerce_ucp/schemas/`.
    - [ ] `ucp_product.json`
    - [ ] `ucp_checkout.json`
    - [ ] `ucp_order.json`
    - [ ] `ap2_mandate.json`
- [ ] Create `bizdefs/commerce_ucp/bizdef.json` with roles and context.
- [ ] Add initial test data in `bizdefs/commerce_ucp/data/test_data.json`.

## Phase 2: A2UI Enhancements
- [ ] Register new component types in `core/api/ui/js/a2ui/catalog.js`.
    - [ ] `ucp-product-card`
    - [ ] `ucp-checkout-summary`
    - [ ] `ap2-mandate-banner`
- [ ] Update `A2UIRenderer` logic if needed (likely just catalog updates).

## Phase 3: Service & Tooling
- [ ] Create `CommerceService` in `core/services/commerce_service.go` to wrap UCP/AP2 logic.
- [ ] Implement `agentic` tools for commerce actions:
    - `commerce_search_products`
    - `commerce_create_checkout`
    - `commerce_verify_mandate`

## Phase 4: Validation
- [ ] Integration tests for `CommerceService`.
- [ ] End-to-end flow from Agent prompt to A2UI rendering.
