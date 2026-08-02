---
page_title: "Testing - Anecdotes Provider Guide"
---

# Testing Guide

## Setup

### Prerequisites

- Go 1.25+
- Terraform 1.0+ on your PATH
- An Anecdotes API key with **Admin** role

> Acceptance tests create and modify real objects in the Anecdotes tenant the
> API key points at. Always run them against a dedicated test tenant, never a
> production one.

### Environment

```bash
export ANECDOTES_API_KEY="your-api-key"
```

Set `TF_ACC=1` only when running acceptance tests, as shown below — unit tests
must run without it.

### Running Tests

```bash
# Unit tests only (no tenant required; TF_ACC unset)
go test ./...

# Run a single acceptance test
TF_ACC=1 go test -v -run TestAccFrameworkResource_create -timeout 60m ./internal/provider/

# Run all acceptance tests for a resource
TF_ACC=1 go test -v -run TestAccControlResource -timeout 60m ./internal/provider/

# Run all acceptance tests for a data source
TF_ACC=1 go test -v -run TestAccControlsDataSource -timeout 60m ./internal/provider/

# Run the full acceptance suite (slow — avoid during development)
TF_ACC=1 go test -v -timeout 120m ./internal/provider/
```

> **Tip**: Always run targeted tests during development.

### Continuous integration

Pull requests run the build, vet, unit tests, lint and the documentation
check. Acceptance tests are not part of that job — run the full suite locally
before a release.

---

## Test Coverage

- **Acceptance tests** (`TF_ACC=1`, live tenant) cover all 7 resources
  (create / update / import, where applicable) and all 11 data sources
  (singular lookups and plural list/filter). They are skipped automatically
  when `TF_ACC` is unset.
- **Unit tests** (no tenant) cover plan-time enum validation, API error
  classification and diagnostics, optional-pointer helpers, and a source-wide
  audit that resources surface API errors through the shared helpers.

---

## Test Helpers

Test helpers are in `internal/provider/provider_test_helpers_test.go`:

| Helper | Purpose |
|--------|---------|
| `testAccPreCheck(t)` | Validates `ANECDOTES_API_KEY` is set |
| `testAccProtoV6ProviderFactories` | Provider factory for acceptance tests |
| `randomName(prefix)` | Generates unique resource names (`prefix-XXXXXX`) |
| `testCheckTotalCountGreaterThan(resource, min)` | Asserts `total_count > min` |
| `testCheckListCountMatchesTotalCount(resource, listAttr)` | Verifies list length matches `total_count` |
| `testAccFrameworkConfig(name)` | Generates a framework config |
| `testAccControlCategoryConfig(fw, cat)` | Generates framework + category config |
| `testAccControlConfig(fw, cat, ctrl)` | Generates framework + category + control config |
| `testAccRequirementConfig(name)` | Generates a requirement config |
| `testAccEvidenceID()` / `testAccEvidencePreCheck(t)` | Read a pre-existing evidence id from `ANECDOTES_TEST_EVIDENCE_ID` |

---

## Writing New Tests

Follow this pattern:

```go
func TestAccWidgetResource_create(t *testing.T) {
    name := randomName("widget")
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                Config: fmt.Sprintf(`
resource "anecdotes_widget" "test" {
  name = %q
}`, name),
                Check: resource.ComposeAggregateTestCheckFunc(
                    resource.TestCheckResourceAttr("anecdotes_widget.test", "name", name),
                    resource.TestCheckResourceAttrSet("anecdotes_widget.test", "widget_id"),
                ),
            },
        },
    })
}
```

For import tests, add a second step:

```go
{
    ResourceName:      "anecdotes_widget.test",
    ImportState:       true,
    ImportStateVerify: true,
    ImportStateVerifyIgnore: []string{"computed_only_field"},
}
```
