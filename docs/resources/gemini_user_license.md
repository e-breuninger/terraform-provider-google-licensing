---
page_title: "googleenterpriselicense_gemini_user_license Resource"
description: |-
  Assigns a Gemini Enterprise or NotebookLM Enterprise license to a user.
---

# googleenterpriselicense_gemini_user_license

Assigns a Gemini Enterprise or NotebookLM Enterprise license to a user via the
[Discovery Engine API](https://cloud.google.com/generative-ai-app-builder/docs/reference/rest).

All arguments force replacement — there is no in-place update.

> **Note:** The Discovery Engine API supports **one active license per user**. Assigning a
> second license config to the same user will overwrite the first. Manage only one
> `googleenterpriselicense_gemini_user_license` resource per user.

## Example Usage

```hcl
resource "googleenterpriselicense_gemini_user_license" "alice_gemini" {
  project        = "my-gcp-project"
  location       = "eu"
  user_id        = "alice@example.com"
  license_config = "projects/123456789/locations/eu/licenseConfigs/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

### Finding your `license_config`

Run the following command to list available license configs in your project:

```shell
curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  "https://eu-discoveryengine.googleapis.com/v1/projects/PROJECT_NUMBER/locations/eu/licenseConfigs"
```

Replace `eu` with your location (`global` or `us`) as appropriate.

## Argument Reference

- `project` (Required, Forces new) — The GCP project ID that holds the Gemini Enterprise subscription.

- `location` (Required, Forces new) — Multi-region for the license. Valid values: `global`, `us`, `eu`.

- `user_id` (Required, Forces new) — The user's primary email address.

- `license_config` (Required, Forces new) — Full resource path of the license configuration:
  `projects/PROJECT_NUMBER/locations/LOCATION/licenseConfigs/SUBSCRIPTION_ID`.

## Attributes Reference

- `id` — Composite identifier: `<project>/<location>/<user_id>`.
- `state` — License assignment state returned by the API (e.g. `ASSIGNED`).

## Import

Import an existing assignment using the composite ID:

```shell
terraform import googleenterpriselicense_gemini_user_license.alice \
  "my-gcp-project/eu/alice@example.com"
```
