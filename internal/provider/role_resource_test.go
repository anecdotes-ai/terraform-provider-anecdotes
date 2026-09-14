// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// mapRoleToState must never overwrite Permissions from the API: a role's
// submitted permissions are echoed but not persisted or enforced, and the
// resolved (inheritance-expanded) set List/Get return is not the same value —
// writing it into a Required attribute broke every apply of a role with a
// non-empty extends. This pins that Permissions is left exactly as it was
// going in, regardless of what the API returns.
func TestMapRoleToState_DoesNotOverwritePermissions(t *testing.T) {
	ctx := context.Background()
	configured, diags := types.SetValueFrom(ctx, types.StringType, []string{})
	if diags.HasError() {
		t.Fatalf("building configured set: %v", diags)
	}

	data := &RoleResourceModel{Permissions: configured}

	role := &client.Role{
		Key:         "cst_00000000_demorole",
		Name:        "DemoRole",
		Description: "d",
		// The resolved, inheritance-expanded set — deliberately different from
		// (and much larger than) what was configured above.
		Permissions: []string{"control:read", "evidence:read", "user:read"},
		Extends:     []string{"viewer_role"},
	}

	var d diag.Diagnostics
	mapRoleToState(ctx, role, data, &d)
	if d.HasError() {
		t.Fatalf("mapRoleToState: %v", d)
	}

	if !data.Permissions.Equal(configured) {
		t.Errorf("Permissions must stay exactly as configured, got %v, want %v", data.Permissions, configured)
	}
}

// The resolved set must still be surfaced somewhere — as EffectivePermissions.
func TestMapRoleToState_SetsEffectivePermissionsFromResolvedSet(t *testing.T) {
	ctx := context.Background()
	data := &RoleResourceModel{}

	role := &client.Role{
		Key:         "cst_00000000_demorole",
		Name:        "DemoRole",
		Permissions: []string{"control:read", "evidence:read", "user:read"},
		Extends:     []string{"viewer_role"},
	}

	var d diag.Diagnostics
	mapRoleToState(ctx, role, data, &d)
	if d.HasError() {
		t.Fatalf("mapRoleToState: %v", d)
	}

	var got []string
	if diags := data.EffectivePermissions.ElementsAs(ctx, &got, false); diags.HasError() {
		t.Fatalf("reading effective_permissions: %v", diags)
	}
	if len(got) != 3 {
		t.Errorf("expected the resolved 3-entry set in effective_permissions, got %v", got)
	}
}

// mapRoleToState must never let a re-fetched CreatedAt overwrite an already
// -known one. Confirmed live: the platform bumps created_at on every
// PUT /roles, the same as updated_at — not a stable creation timestamp
// server-side. created_at carries UseStateForUnknown, so Update's plan
// already holds the prior known value; if mapRoleToState blindly copied the
// freshly re-fetched (now-bumped) value in, Terraform would report
// "provider produced inconsistent result after apply" on every update — which
// is exactly what happened before this was fixed (caught by
// TestAccRoleResource_update against a live tenant, not by this test, since a
// pure unit test can't see the platform bump anywhere but here).
func TestMapRoleToState_PreservesCreatedAtAcrossUpdate(t *testing.T) {
	ctx := context.Background()
	data := &RoleResourceModel{CreatedAt: types.StringValue("2026-01-01T00:00:00Z")}

	role := &client.Role{
		Key:  "cst_00000000_demorole",
		Name: "DemoRole",
		// The platform's write-time bookkeeping value — deliberately later
		// than, and different from, the already-known state value above.
		CreatedAt: "2026-01-02T00:00:00Z",
		UpdatedAt: "2026-01-02T00:00:00Z",
	}

	var d diag.Diagnostics
	mapRoleToState(ctx, role, data, &d)
	if d.HasError() {
		t.Fatalf("mapRoleToState: %v", d)
	}

	if data.CreatedAt.ValueString() != "2026-01-01T00:00:00Z" {
		t.Errorf("CreatedAt must stay as it was in state, got %q", data.CreatedAt.ValueString())
	}
	if data.UpdatedAt.ValueString() != "2026-01-02T00:00:00Z" {
		t.Errorf("UpdatedAt should always reflect the fresh value, got %q", data.UpdatedAt.ValueString())
	}
}

// A config carrying `full_access_frameworks = []` must survive an apply. The
// platform normalizes an empty list to JSON null and can never echo [] back, so
// writing null into state here would fail the apply with "provider produced
// inconsistent result after apply" against a plan holding [].
func TestMapRoleToState_PreservesConfiguredEmptyFullAccessFrameworks(t *testing.T) {
	ctx := context.Background()
	emptyList, d := types.ListValueFrom(ctx, types.StringType, []string{})
	if d.HasError() {
		t.Fatalf("build empty list: %v", d)
	}
	data := &RoleResourceModel{FullAccessFrameworks: emptyList}

	// Unscoped role: the platform reports null, not [].
	role := &client.Role{Key: "cst_00000000_demorole", Name: "DemoRole", FullAccessFrameworks: nil}

	var diags diag.Diagnostics
	mapRoleToState(ctx, role, data, &diags)
	if diags.HasError() {
		t.Fatalf("mapRoleToState: %v", diags)
	}

	if data.FullAccessFrameworks.IsNull() {
		t.Fatal("a configured empty list must not be replaced by null")
	}
	if n := len(data.FullAccessFrameworks.Elements()); n != 0 {
		t.Errorf("expected the empty list to be preserved, got %d elements", n)
	}
}

// The converse: when the attribute was never configured, an unscoped role still
// has to land as null rather than an invented empty list.
func TestMapRoleToState_UnsetFullAccessFrameworksStaysNull(t *testing.T) {
	ctx := context.Background()
	data := &RoleResourceModel{FullAccessFrameworks: types.ListNull(types.StringType)}

	role := &client.Role{Key: "cst_00000000_demorole", Name: "DemoRole", FullAccessFrameworks: nil}

	var diags diag.Diagnostics
	mapRoleToState(ctx, role, data, &diags)
	if diags.HasError() {
		t.Fatalf("mapRoleToState: %v", diags)
	}

	if !data.FullAccessFrameworks.IsNull() {
		t.Errorf("expected null, got %v", data.FullAccessFrameworks)
	}
}

// Drift must still be reported: a role that had frameworks and no longer does
// comes back as null even though state held a value.
func TestMapRoleToState_ClearedFullAccessFrameworksBecomesNull(t *testing.T) {
	ctx := context.Background()
	prior, d := types.ListValueFrom(ctx, types.StringType, []string{"790498536_cb1a5c3de0"})
	if d.HasError() {
		t.Fatalf("build list: %v", d)
	}
	data := &RoleResourceModel{FullAccessFrameworks: prior}

	role := &client.Role{Key: "cst_00000000_demorole", Name: "DemoRole", FullAccessFrameworks: nil}

	var diags diag.Diagnostics
	mapRoleToState(ctx, role, data, &diags)
	if diags.HasError() {
		t.Fatalf("mapRoleToState: %v", diags)
	}

	if !data.FullAccessFrameworks.IsNull() {
		t.Errorf("expected the cleared value to surface as null, got %v", data.FullAccessFrameworks)
	}
}

// TestAccRoleResource_create covers the simplest case: no extends, no
// permissions.
func TestAccRoleResource_create(t *testing.T) {
	name := randomName("role")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Acceptance test role"
  permissions = []
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_role.test", "name", name),
					resource.TestCheckResourceAttrSet("anecdotes_role.test", "role_id"),
					resource.TestCheckResourceAttr("anecdotes_role.test", "permissions.#", "0"),
				),
			},
		},
	})
}

// TestAccRoleResource_extends is the regression test for the blocker where
// every apply of a role with a non-empty extends failed with "provider
// produced inconsistent result after apply": the platform always returns the
// inheritance-expanded permission set on read, never an echo of what was
// configured, once extends resolves to anything.
func TestAccRoleResource_extends(t *testing.T) {
	name := randomName("role-extends")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Acceptance test role with extends"
  extends     = ["viewer_role"]
  permissions = []
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_role.test", "name", name),
					resource.TestCheckResourceAttr("anecdotes_role.test", "permissions.#", "0"),
					resource.TestCheckResourceAttr("anecdotes_role.test", "extends.#", "1"),
					// The resolved set inherited from viewer_role is non-empty —
					// this is the value permissions must NOT be set to.
					testCheckCollectionNotEmpty("anecdotes_role.test", "effective_permissions"),
				),
			},
		},
	})
}

func TestAccRoleResource_update(t *testing.T) {
	name := randomName("role-upd")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Original description"
  permissions = []
}`, name),
				Check: resource.TestCheckResourceAttr("anecdotes_role.test", "description", "Original description"),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Updated description"
  permissions = []
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_role.test", "description", "Updated description"),
					resource.TestCheckResourceAttr("anecdotes_role.test", "name", name),
				),
			},
		},
	})
}

// An empty full_access_frameworks must apply cleanly. The platform normalizes
// [] to null and can never echo it back, so before the mapping fix this failed
// the apply outright with "provider produced inconsistent result after apply".
func TestAccRoleResource_emptyFullAccessFrameworks(t *testing.T) {
	name := randomName("role-faf")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name                   = %q
  description            = "Empty full_access_frameworks"
  full_access_frameworks = []
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_role.test", "name", name),
					resource.TestCheckResourceAttr("anecdotes_role.test", "full_access_frameworks.#", "0"),
				),
			},
		},
	})
}

// extends = [] is rejected at plan time rather than failing the apply: the
// platform substitutes ["basic_role"] for an empty extends exactly as it does
// for an omitted one, so the configured value could never be honored.
func TestAccRoleResource_emptyExtendsRejected(t *testing.T) {
	name := randomName("role-ext0")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Empty extends"
  extends     = []
}`, name),
				ExpectError: regexp.MustCompile(`(?s)extends.*at least 1`),
			},
		},
	})
}

// Removing an Optional+Computed attribute from config must actually remove it.
// Without the reset plan modifier the prior value is carried forward into the
// plan and re-sent on the full-object PUT, so Terraform reports "No changes"
// and the role silently keeps the inheritance the user just deleted.
func TestAccRoleResource_removingExtendsResetsToDefault(t *testing.T) {
	name := randomName("role-extdel")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Extends removal"
  extends     = ["viewer_role"]
}`, name),
				Check: resource.TestCheckTypeSetElemAttr("anecdotes_role.test", "extends.*", "viewer_role"),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Extends removal"
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Back to the platform default, not the removed value.
					resource.TestCheckTypeSetElemAttr("anecdotes_role.test", "extends.*", "basic_role"),
					resource.TestCheckResourceAttr("anecdotes_role.test", "extends.#", "1"),
				),
			},
		},
	})
}

// A role imported from a config that does not set permissions must plan clean.
// permissions is deliberately never read back from the platform, so it lands
// null on import; while it was Required that guaranteed a permanent diff.
func TestAccRoleResource_importPlansClean(t *testing.T) {
	name := randomName("role-impclean")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Import clean-plan test"
}`, name),
			},
			{
				ResourceName:                         "anecdotes_role.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "role_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_role.test", "role_id"),
			},
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Import clean-plan test"
}`, name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccRoleResource_import(t *testing.T) {
	name := randomName("role-imp")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_role" "test" {
  name        = %q
  description = "Import test"
  permissions = []
}`, name),
			},
			{
				ResourceName:                         "anecdotes_role.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "role_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_role.test", "role_id"),
			},
		},
	})
}
