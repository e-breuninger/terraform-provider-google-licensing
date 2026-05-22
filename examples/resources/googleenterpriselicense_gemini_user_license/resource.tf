# Assign a Gemini Enterprise (or NotebookLM Enterprise) license to a user
resource "googleenterpriselicense_gemini_user_license" "example" {
  project        = "my-gcp-project"
  location       = "eu" # "global", "us", or "eu"
  user_id        = "alice@example.com"
  license_config = "projects/123456789/locations/eu/licenseConfigs/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
