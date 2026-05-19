# Specification: Commerce UCP & AP2 Integration

## 1. Goal
Integrate the Universal Commerce Protocol (UCP) and Agent Payments Protocol (AP2) into AiGenApp. This allows agents to handle commerce workflows (discovery, checkout, payments) natively, with rich A2UI visualization for protocol-specific data.

## 2. Business Logic (BizDef)
The `commerce_ucp` BizDef will model the core entities required for UCP and AP2 workflows.

### Entities:
- **`ucp_product`**: Product catalog data (mapped from UCP Product model).
- **`ucp_checkout`**: Active checkout sessions (mapped from UCP Checkout model).
- **`ucp_order`**: Finalized orders (mapped from UCP Order model).
- **`ap2_mandate`**: Cryptographic authorizations for intents or carts.

## 3. A2UI Display Logic
Implement specialized rendering for UCP/AP2 JSON outputs to make them "Agent-Native" and human-readable.

### Components:
- **`UcpProductCard`**: Visual representation of a product with images, price, and "Add to Cart" action.
- **`UcpCheckoutSummary`**: Overview of items, total, and AP2 mandate status.
- **`Ap2MandateBanner`**: A security banner showing verification status of a mandate.

## 4. Integration with `ucp-srv`
- AiGenApp will act as a consumer/proxy for `ucp-srv` (MCP).
- Tools will be exposed via the `agentic` framework to interact with `ucp-srv`.
