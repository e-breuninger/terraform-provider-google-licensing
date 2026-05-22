---
page_title: "googleenterpriselicense_assignment Resource"
description: |-
  Assigns a Google Workspace license SKU to a user.
---

# googleenterpriselicense_assignment

Assigns a Google Workspace (Enterprise) license SKU to a user via the
[Enterprise License Manager API](https://developers.google.com/admin-sdk/licensing/reference/rest/v1/licenseAssignments).

Changing `sku_id` performs an in-place PATCH (no revoke / re-assign).
Changing `product_id` or `user_id` forces replacement.

## Example Usage

```hcl
resource "googleenterpriselicense_assignment" "alice" {
  product_id = "Google-Apps"
  sku_id     = "1010020027" # Google Workspace Business Starter
  user_id    = "alice@example.com"
}
```

### Change SKU in-place

```hcl
resource "googleenterpriselicense_assignment" "alice" {
  product_id = "Google-Apps"
  sku_id     = "1010060001" # Google Workspace Business Plus
  user_id    = "alice@example.com"
}
```

## Argument Reference

- `product_id` (Required, Forces new) — The product's unique identifier. Common values:
  - `Google-Apps` — Google Workspace
  - `Google-Vault` — Google Vault
  - `Google-Drive-storage` — Google Drive storage

  See the [products and SKUs reference](https://developers.google.com/admin-sdk/licensing/v1/how-tos/products).

- `sku_id` (Required) — The SKU's unique identifier within the product. Changing this value performs a PATCH (in-place SKU upgrade/downgrade) without revoking the assignment.

- `user_id` (Required, Forces new) — The user's primary email address or unique user ID.

## Attributes Reference

- `id` — Composite identifier: `<product_id>/<sku_id>/<user_id>`.
- `product_name` — Human-readable display name of the product.
- `sku_name` — Human-readable display name of the SKU.

## Import

Import an existing assignment using the composite ID:

```shell
terraform import googleenterpriselicense_assignment.alice \
  "Google-Apps/1010020027/alice@example.com"
```
