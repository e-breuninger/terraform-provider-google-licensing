# Assign a Google Workspace Business Starter license
resource "googleenterpriselicense_assignment" "example" {
  product_id = "Google-Apps"
  sku_id     = "1010020027" # Google Workspace Business Starter
  user_id    = "alice@example.com"
}
