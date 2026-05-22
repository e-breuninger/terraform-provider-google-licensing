---
page_title: "Google Enterprise License Provider"
description: |-
  Manage Google Workspace and Gemini Enterprise licenses with Terraform.
---

# Google Enterprise License Provider

The **googleenterpriselicense** provider manages Google Workspace license assignments via the
[Enterprise License Manager API](https://developers.google.com/admin-sdk/licensing) and
Gemini Enterprise / NotebookLM Enterprise licenses via the
[Discovery Engine API](https://cloud.google.com/generative-ai-app-builder/docs/reference/rest).

## Authentication

The provider supports three credential strategies, evaluated in order:

1. **`credentials`** — inline JSON of a service-account key.
2. **`credentials_file`** — path to a JSON credentials file.
3. **Application Default Credentials (ADC)** — used automatically when neither of the above is set.

### Required OAuth scopes

| API | Scope |
|-----|-------|
| Enterprise License Manager | `https://www.googleapis.com/auth/apps.licensing` |
| Discovery Engine (Gemini) | `https://www.googleapis.com/auth/cloud-platform` |

When using ADC with `gcloud`, request both scopes:

```shell
gcloud auth application-default login \
  --scopes=https://www.googleapis.com/auth/apps.licensing,https://www.googleapis.com/auth/cloud-platform
```

### Service account

Grant the service account the following IAM roles:

- **License Management** (`roles/licensemanager.licenseAdmin`) — for `googleenterpriselicense_assignment`
- **Discovery Engine Admin** (`roles/discoveryengine.admin`) — for `googleenterpriselicense_gemini_user_license`

## Example Usage

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
  # Uses ADC when credentials / credentials_file are omitted
}
```

## Schema

### Optional

- `credentials` (String, Sensitive) — JSON content of a Google service-account key file. Can also be set via the `GOOGLE_CREDENTIALS` environment variable.
- `credentials_file` (String) — Path to a JSON credentials file. Can also be set via the `GOOGLE_APPLICATION_CREDENTIALS` environment variable.
