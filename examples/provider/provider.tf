terraform {
  required_providers {
    googleenterpriselicense = {
      source  = "e-breuninger/google-licensing"
      version = "~> 0.1"
    }
  }
}

# Option 1: inline service-account JSON
provider "googleenterpriselicense" {
  credentials = file("service-account.json")
}

# Option 2: path to a credentials file
provider "googleenterpriselicense" {
  credentials_file = "/path/to/service-account.json"
}

# Option 3: Application Default Credentials (ADC) — omit both attributes
provider "googleenterpriselicense" {}
