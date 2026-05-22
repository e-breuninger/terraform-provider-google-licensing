# Terraform Provider: Google Enterprise License

Manage Google Workspace and Gemini Enterprise licenses declaratively with Terraform.

| Resource | API |
|----------|-----|
| `googleenterpriselicense_assignment` | [Enterprise License Manager API](https://developers.google.com/admin-sdk/licensing) |
| `googleenterpriselicense_gemini_user_license` | [Discovery Engine API](https://cloud.google.com/generative-ai-app-builder/docs/reference/rest) |

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5
- [Go](https://golang.org/doc/install) >= 1.21 (to build from source)
- A Google Workspace domain with Admin SDK / Enterprise License Manager enabled
- Appropriate IAM permissions (see [Authentication](#authentication))

## Usage

```hcl
terraform {
  required_providers {
    googleenterpriselicense = {
      source  = "e-breuninger/google-licensing"
      version = "~> 0.1"
    }
  }
}

provider "googleenterpriselicense" {
  # Uses Application Default Credentials when omitted.
  # credentials      = file("service-account.json")
  # credentials_file = "/path/to/service-account.json"
}

# Assign a Google Workspace Business Starter license
resource "googleenterpriselicense_assignment" "alice" {
  product_id = "Google-Apps"
  sku_id     = "1010020027"
  user_id    = "alice@example.com"
}

# Assign a Gemini Enterprise license
resource "googleenterpriselicense_gemini_user_license" "alice_gemini" {
  project        = "my-gcp-project"
  location       = "eu"
  user_id        = "alice@example.com"
  license_config = "projects/123456789/locations/eu/licenseConfigs/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

## Authentication

The provider evaluates credentials in this order:

1. **`credentials`** -- inline JSON of a service-account key (or set `GOOGLE_CREDENTIALS`).
2. **`credentials_file`** -- path to a credentials file (or set `GOOGLE_APPLICATION_CREDENTIALS`).
3. **Application Default Credentials (ADC)** -- picked up automatically via `gcloud auth application-default login`.

### Required OAuth scopes

| Resource | Scope |
|----------|-------|
| `googleenterpriselicense_assignment` | `https://www.googleapis.com/auth/apps.licensing` |
| `googleenterpriselicense_gemini_user_license` | `https://www.googleapis.com/auth/cloud-platform` |

When using ADC, request both scopes at login time:

```shell
gcloud auth application-default login \
  --scopes=https://www.googleapis.com/auth/apps.licensing,https://www.googleapis.com/auth/cloud-platform
```

### Service account permissions

- **License Management** -- Admin SDK > Enterprise License Manager (or `roles/licensemanager.licenseAdmin`)
- **Gemini licenses** -- Discovery Engine Admin (`roles/discoveryengine.admin`)

## Resources

### `googleenterpriselicense_assignment`

Assigns a Google Workspace SKU to a user. Changing `sku_id` performs an in-place PATCH
(no revoke/re-assign). Changing `product_id` or `user_id` forces replacement.

```hcl
resource "googleenterpriselicense_assignment" "example" {
  product_id = "Google-Apps"
  sku_id     = "1010020027"   # Google Workspace Business Starter
  user_id    = "user@example.com"
}
```

**Import:**
```shell
terraform import googleenterpriselicense_assignment.example \
  "Google-Apps/1010020027/user@example.com"
```

### `googleenterpriselicense_gemini_user_license`

Assigns a Gemini Enterprise or NotebookLM Enterprise license to a user.

> **One license per user:** The Discovery Engine API supports only one active license
> config per user. Do not create two resources for the same user with different
> `license_config` values -- each apply will overwrite the other.

```hcl
resource "googleenterpriselicense_gemini_user_license" "example" {
  project        = "my-gcp-project"
  location       = "eu"                    # "global", "us", or "eu"
  user_id        = "user@example.com"
  license_config = "projects/PROJECT_NUMBER/locations/LOCATION/licenseConfigs/SUBSCRIPTION_ID"
}
```

**Finding your `license_config`:**
```shell
curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  "https://eu-discoveryengine.googleapis.com/v1/projects/PROJECT_NUMBER/locations/eu/licenseConfigs"
```

**Import:**
```shell
terraform import googleenterpriselicense_gemini_user_license.example \
  "my-gcp-project/eu/user@example.com"
```

## Building from Source

```shell
git clone https://github.com/e-breuninger/terraform-provider-google-licensing
cd terraform-provider-google-licensing
make build
```

### Local development override

Add this to `~/.terraformrc` to use your local build without publishing to the registry:

```hcl
provider_installation {
  dev_overrides {
    "e-breuninger/google-licensing" = "/path/to/terraform-provider-google-licensing"
  }
  direct {}
}
```

Then run `terraform plan` / `terraform apply` directly -- **skip `terraform init`** when dev overrides are active.

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com) and GitHub Actions.

1. Tag the commit: `git tag v0.1.0 && git push origin v0.1.0`
2. The [release workflow](.github/workflows/release.yml) builds multi-platform binaries, signs the checksums with GPG, and creates a GitHub Release.
3. To publish to the [Terraform Registry](https://registry.terraform.io), connect the repository there and configure the `GPG_PRIVATE_KEY` / `GPG_PASSPHRASE` secrets.

## License

[Mozilla Public License v2.0](LICENSE)
