# Project Agents Guide

## Goal
The goal of this project is to reimplement the legacy `flarectl` command-line tool using the modern `github.com/cloudflare/cloudflare-go/v6` library. The new tool is tentatively named `flarectl6` (or just `flarectl` in the project context).

## Context
- **Legacy Tool**: The original `flarectl` is located in `ref/flarectl` (if fetched). It uses an older version of the Cloudflare API and library.
- **New Library**: The `cloudflare-go/v6` library is located in `ref/cloudflare-go` (if fetched). It supports newer features and has a different API structure compared to what the legacy tool used.

## Workflow
1. **Analyze**: Look at a command in `ref/flarectl` to understand its flags, arguments, and intended behavior.
2. **Map**: Find the equivalent functionality in `ref/cloudflare-go`. Note differences in API calls, types, and logic.
3. **Implement**: Create the equivalent command in the new project using Cobra, calling the v6 library.
4. **Document**: Update `doc/` as we learn more about the mapping.

## detailed References
- `ref/flarectl`: Source code for the behavior we want to replicate.
- `ref/cloudflare-go`: Source code for the library we must use.
- `README.md`: Setup instructions.

## Coding Standards
- Use `cobra` for CLI commands.
- Check errors explicitly.
- Follow Go conventions.

## Gotchas
- `ref/` directory contains legacy code without `go.mod`. This can cause `go vet` and `go mod tidy` to process it if run from the root.
  - **Workaround**: Run `cd ref && go mod init ref` to isolate it, or exclude it from tool runs. `ref/` is gitignored.
- **Legacy `dns create-or-update` behavior**: The legacy tool searches for records by FQDN. If *any* records exist (e.g. `MX` records), it enters an update loop. If the requested type (e.g. `A`) is not found in that loop, it does *nothing* (neither updates nor creates). This behavior was replicated to satisfy "Zero Regression", but it may be unintuitive (preventing creation of new types on existing names).

## Learnings (New)
- **Library Version**: `cloudflare-go` v6 is generated via Stainless and behaves differently from v4 (legacy).
  - Use `client.Zones.ListAutoPaging` for iterating over all results.
  - Use `cloudflare.F()` helper to wrap parameters.
  - `List` returns a struct with a `Result` slice, but the iterator (`ListAutoPaging`) is preferred for complete lists.
- **Tablewriter**: `tablewriter` v1.1.3 has a breaking API change. We pinned v0.0.5 to match legacy behavior and simplify porting.
- **DNS Records**:
  - `dns list` and `dns create` use different output table columns/order in legacy. Replicated this inconsistency.
  - `dns update` uses `PATCH` (Edit) semantics in v6. Implemented using `c.Flags().Changed()` to only send updated fields.
- **User Command**:
  - `user info`: v6 `UserGetResponse` structure differs significantly from legacy. Fields `Email` and `Username` are missing in v6 struct. Output columns are preserved but values are empty.
  - `user update`: Legacy command is a no-op (returns `nil`). Replicated this behavior.
- **User-Agent Command**:
  - `user-agents list`: Implemented `page` flag. Legacy behavior defaults to page 1 if not specified, so we preserved this (using `List` instead of `ListAutoPaging` when page is involved).
  - `user-agents update`: `Target` must be set to "ua". The v6 library's `UARuleUpdateParamsConfigurationTarget` constants do not include "ua", but the API requires it (and legacy used it). We cast the string "ua" to the target type.
- **Firewall Command**:
  - `firewall access-rules`: v6 `AccessRuleService` does not support User-level scope (no `ListUserAccessRules` or equivalent in generated SDK). Commands without `--zone` or `--account` will return an error, whereas legacy would default to user scope.
- **IPs Command**:
  - `ips`: Replicated a legacy quirk where the `--ip-only` flag *enables* the header output ("IPv4 ranges:"), contrary to the flag name.
- **PageRules Command**:
  - `pagerules`: `PageRuleAction.Value` is a union interface in v6. Replicated legacy formatting logic by inspecting `ID` and type switching.
  - Implemented a custom `title` helper to replace deprecated `strings.Title` without adding new dependencies.
- **Origin CA Root Cert Command**:
  - `origin-ca-root-cert`: v6 library lacks a helper for this. Implemented via manual `client.Get` to `/cert_req`.
- **Railgun Command**:
  - `railgun`: Implemented as a no-op placeholder, matching legacy behavior.
- **Metrics**:
  - `cmd/zone.go` implemented (~160 LOC).
  - `cmd/dns.go` implemented (~320 LOC).
  - `cmd/utils.go` added (~70 LOC).
  - `cmd/user_agent.go` implemented (~180 LOC).
  - `cmd/firewall.go` implemented (~380 LOC).
  - `cmd/ips.go` implemented (~70 LOC).
  - `cmd/pagerules.go` implemented (~140 LOC).
  - `cmd/origin_ca_root_cert.go` implemented (~60 LOC).
  - Complexity remains low.
