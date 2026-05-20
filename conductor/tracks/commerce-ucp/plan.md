# Implementation Plan: Commerce UCP & AP2 Integration

## Phase 1: BizDef Definition
- [x] Define `commerce_ucp` entities in `bizdefs/commerce_ucp/schemas/`.
    - [x] `ucp_product.json`
    - [x] `ucp_checkout.json`
    - [x] `ucp_order.json`
    - [x] `ap2_mandate.json`
- [x] Create `bizdefs/commerce_ucp/bizdef.json` with roles and context.
- [x] Add initial test data in `bizdefs/commerce_ucp/data/test_data.json`.

## Phase 2: A2UI Enhancements
- [x] Register new component types in `core/api/ui/js/a2ui/catalog.js`.
    - [x] `ucp-product-card` (as `UcpProductCard`)
    - [x] `ucp-checkout-summary` (as `UcpCheckoutSummary`)
    - [x] `ap2-mandate-banner` (as `Ap2MandateBanner`)
- [x] Update `A2UIRenderer` logic if needed (likely just catalog updates).

## Phase 3: Service & Tooling
- [x] Create `CommerceService` in `core/services/commerce_service.go` to wrap UCP/AP2 logic.
- [ ] Implement `agentic` tools for commerce actions:
    - [x] `commerce_search_products` (implemented as `cms_commerce_search`)
    - [x] `commerce_create_checkout` (implemented as `cms_commerce_checkout`)
    - [ ] `commerce_verify_mandate`
- [ ] Create `CommerceApi` in `core/api/commerce_api.go` for direct access.

## Phase 4: Validation
- [x] Integration tests for `CommerceService`.
- [ ] End-to-end flow from Agent prompt to A2UI rendering.
- [ ] Verify `ap2_mandate` verification logic with mock signatures.
