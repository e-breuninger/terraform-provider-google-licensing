# Changelog

All notable changes to this provider will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-07-22

### Added

- `googleenterpriselicense_assignment` resource — assign and manage Google Workspace SKU licenses via the Enterprise License Manager API. Supports in-place SKU changes via PATCH and import.
- `googleenterpriselicense_gemini_user_license` resource — assign Gemini Enterprise / NotebookLM Enterprise licenses via the Discovery Engine API. Supports import.
- Provider authentication via inline credentials JSON, credentials file path, or Application Default Credentials (ADC).

### Fixed

- `googleenterpriselicense_gemini_user_license` no longer removes a resource from state when the assigned license config drifts from configuration; it now reflects the actual license so drift surfaces as a plan diff instead of silent state loss.

### Changed

- Lowered the `go.mod` toolchain directive from `go 1.25.8` to `go 1.25` to avoid forcing an exact patch-level Go toolchain download.
