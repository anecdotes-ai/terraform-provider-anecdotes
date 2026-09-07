// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testCheckPlaybookStepOnPlatform reads the playbook straight from the API
// (bypassing Terraform state) to confirm a step field was persisted by the
// platform, not just echoed into local state.
func testCheckPlaybookStepOnPlatform(t *testing.T, resourceName string, index int, want map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := stateAttr(s, resourceName, "playbook_id")
		if err != nil {
			return err
		}
		playbook, err := testAccNewClient(t).GetPlaybook(context.Background(), id)
		if err != nil {
			return fmt.Errorf("reading playbook %s: %w", id, err)
		}
		if index >= len(playbook.Steps) {
			return fmt.Errorf("playbook %s has %d steps, wanted step %d", id, len(playbook.Steps), index)
		}

		step := playbook.Steps[index]
		got := map[string]string{
			"step_title":          step.StepTitle,
			"step_trigger_event":  step.StepTriggerEvent,
			"step_action_type":    step.StepActionType,
			"step_url_to_trigger": step.StepURLToTrigger,
		}
		if payload, err := json.Marshal(step.PayloadConfiguration); err == nil {
			got["payload_configuration"] = string(payload)
		}

		for field, wantValue := range want {
			if got[field] != wantValue {
				return fmt.Errorf("step %d %s on the platform is %q, expected %q", index, field, got[field], wantValue)
			}
		}
		return nil
	}
}

func testAccPlaybookConfig(title, stepTitle, url string) string {
	return fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "acceptance test playbook"

  steps = [
    {
      title          = %q
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = %q

      payload_configuration = jsonencode({ source = "terraform" })
    },
  ]
}`, title, stepTitle, url)
}

func TestAccPlaybookResource_createUpdateImport(t *testing.T) {
	title := randomName("pb")
	updated := randomName("pb-updated")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaybookConfig(title, "first step", "https://example.com/tf-acc"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("anecdotes_playbook.test", "playbook_id"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "title", title),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "active", "true"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "type", "playbook"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.#", "1"),
					resource.TestCheckResourceAttrSet("anecdotes_playbook.test", "steps.0.step_id"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.0.internal_action", "false"),
					// The platform, not local state, must hold the configured values.
					testCheckPlaybookStepOnPlatform(t, "anecdotes_playbook.test", 0, map[string]string{
						"step_title":            "first step",
						"step_action_type":      "webhook",
						"step_url_to_trigger":   "https://example.com/tf-acc",
						"payload_configuration": `{"source":"terraform"}`,
					}),
				),
			},
			{
				// Editing a step in place keeps the playbook: only its
				// membership is fixed.
				Config: testAccPlaybookConfig(updated, "renamed step", "https://example.com/tf-acc-2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_playbook.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "title", updated),
					testCheckPlaybookStepOnPlatform(t, "anecdotes_playbook.test", 0, map[string]string{
						"step_title":          "renamed step",
						"step_url_to_trigger": "https://example.com/tf-acc-2",
					}),
				),
			},
			{
				ResourceName:                         "anecdotes_playbook.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "playbook_id",
				ImportStateIdFunc:                    importIDFromAttr("anecdotes_playbook.test", "playbook_id"),
			},
		},
	})
}

// A step that is edited comes back last from the API. Read must restore the
// configured order, otherwise the next plan reports a change that is not one.
func TestAccPlaybookResource_stepOrderSurvivesUpdate(t *testing.T) {
	title := randomName("pb-order")

	config := func(firstStepTitle string) string {
		return fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "step ordering"

  steps = [
    {
      step_id        = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"
      title          = %q
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-1"
    },
    {
      step_id        = "3f2504e0-4f89-41d3-9a0c-0305e82c3302"
      title          = "second"
      trigger_event  = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-2"
    },
    {
      step_id        = "3f2504e0-4f89-41d3-9a0c-0305e82c3303"
      title          = "third"
      trigger_event  = "3f2504e0-4f89-41d3-9a0c-0305e82c3302"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-3"
    },
  ]
}`, title, firstStepTitle)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.0.title", "first"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.1.title", "second"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.2.title", "third"),
				),
			},
			{
				// Renaming the first step moves it last in the API response.
				Config: config("first renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.0.title", "first renamed"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.1.title", "second"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.2.title", "third"),
				),
			},
			{
				// The reordering must not surface as a pending change.
				Config:   config("first renamed"),
				PlanOnly: true,
			},
		},
	})
}

// Steps cannot be added or removed on an existing playbook, so a change in
// membership must plan as a replacement rather than an update.
func TestAccPlaybookResource_addingAStepReplaces(t *testing.T) {
	title := randomName("pb-steps")

	twoSteps := fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "step membership"

  steps = [
    {
      title          = "first"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-1"
    },
    {
      title          = "second"
      trigger_event  = "EvidenceGapDetected"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-2"
    },
  ]
}`, title)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaybookConfig(title, "first", "https://example.com/tf-acc-1"),
				Check:  resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.#", "1"),
			},
			{
				Config: twoSteps,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_playbook.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.#", "2"),
			},
		},
	})
}

func TestAccPlaybookResource_disabled(t *testing.T) {
	title := randomName("pb-inactive")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "created disabled"
  active      = false

  steps = [
    {
      title          = "first"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc"
    },
  ]
}`, title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "active", "false"),
					func(s *terraform.State) error {
						id, err := stateAttr(s, "anecdotes_playbook.test", "playbook_id")
						if err != nil {
							return err
						}
						playbook, err := testAccNewClient(t).GetPlaybook(context.Background(), id)
						if err != nil {
							return fmt.Errorf("reading playbook %s: %w", id, err)
						}
						if playbook.Active {
							return fmt.Errorf("playbook %s is active on the platform, expected it disabled", id)
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccPlaybookResource_scheduled(t *testing.T) {
	title := randomName("pb-scheduled")

	config := func(playbookTitle string) string {
		return fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "scheduled playbook"

  schedule_config = {
    period     = "week"
    time       = "09:30"
    timezone   = "America/New_York"
    start_date = "2026-09-07T00:00:00Z"
    ends_in    = "month"
  }

  steps = [
    {
      title          = "scheduled step"
      trigger_event  = "ScheduledPlaybookTriggered"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-scheduled"
    },
  ]
}`, playbookTitle)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(title),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "schedule_config.period", "week"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "schedule_config.time", "09:30"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "schedule_config.timezone", "America/New_York"),
					// end_date is derived from start_date and ends_in.
					resource.TestCheckResourceAttrSet("anecdotes_playbook.test", "schedule_config.end_date"),
				),
			},
			{
				// The schedule's stored form must not read back as a change.
				Config:   config(title),
				PlanOnly: true,
			},
			{
				// An update must not send the derived end_date back alongside
				// ends_in: the two are mutually exclusive on the wire.
				Config: config(title + "-renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_playbook.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "title", title+"-renamed"),
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "schedule_config.ends_in", "month"),
					resource.TestCheckResourceAttrSet("anecdotes_playbook.test", "schedule_config.end_date"),
				),
			},
		},
	})
}

// A filter keeps empty top-level values, unlike the payload and header
// configurations, so one must not be rejected at plan time.
func TestAccPlaybookResource_filterKeepsEmptyValues(t *testing.T) {
	title := randomName("pb-filter")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "filter with an empty value"

  steps = [
    {
      title          = "filtered step"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-filter"

      filter_configuration = jsonencode({
        left     = "control_status"
        operator = "Is"
        right    = ["IN_PROGRESS"]
        negate   = false
      })
    },
  ]
}`, title),
				Check: func(s *terraform.State) error {
					id, err := stateAttr(s, "anecdotes_playbook.test", "playbook_id")
					if err != nil {
						return err
					}
					playbook, err := testAccNewClient(t).GetPlaybook(context.Background(), id)
					if err != nil {
						return fmt.Errorf("reading playbook %s: %w", id, err)
					}
					if _, ok := playbook.Steps[0].FilterConfiguration["negate"]; !ok {
						return fmt.Errorf("the platform did not keep the empty filter value: %v",
							playbook.Steps[0].FilterConfiguration)
					}
					return nil
				},
			},
		},
	})
}

func TestAccPlaybookResource_internalActionStep(t *testing.T) {
	title := randomName("pb-internal")

	config := func(stepTitle string) string {
		return fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "internal action step"

  steps = [
    {
      title         = %q
      trigger_event = "ControlStatusChanged"
      action_type   = "create_task"
    },
  ]
}`, title, stepTitle)
	}

	checkInternalOnPlatform := func(s *terraform.State) error {
		id, err := stateAttr(s, "anecdotes_playbook.test", "playbook_id")
		if err != nil {
			return err
		}
		playbook, err := testAccNewClient(t).GetPlaybook(context.Background(), id)
		if err != nil {
			return fmt.Errorf("reading playbook %s: %w", id, err)
		}
		if !playbook.Steps[0].InternalAction {
			return fmt.Errorf("step is not marked as an internal action on the platform")
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("internal step"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.0.internal_action", "true"),
					resource.TestCheckNoResourceAttr("anecdotes_playbook.test", "steps.0.url_to_trigger"),
					checkInternalOnPlatform,
				),
			},
			{
				// Editing the step must leave it internal.
				Config: config("internal step renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("anecdotes_playbook.test", "steps.0.internal_action", "true"),
					resource.TestCheckNoResourceAttr("anecdotes_playbook.test", "steps.0.url_to_trigger"),
					checkInternalOnPlatform,
				),
			},
			{
				Config:   config("internal step renamed"),
				PlanOnly: true,
			},
		},
	})
}

// A schedule cannot be removed from a playbook, so dropping schedule_config
// must plan as a replacement rather than an update.
func TestAccPlaybookResource_removingScheduleReplaces(t *testing.T) {
	title := randomName("pb-unschedule")

	scheduled := fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "schedule removal"

  schedule_config = {
    period     = "day"
    time       = "09:00"
    start_date = "2026-09-07T00:00:00Z"
  }

  steps = [
    {
      title          = "step"
      trigger_event  = "ScheduledPlaybookTriggered"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-unschedule"
    },
  ]
}`, title)

	unscheduled := fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "schedule removal"

  steps = [
    {
      title          = "step"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-unschedule"
    },
  ]
}`, title)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: scheduled,
				Check:  resource.TestCheckResourceAttr("anecdotes_playbook.test", "schedule_config.period", "day"),
			},
			{
				Config: unscheduled,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("anecdotes_playbook.test", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckNoResourceAttr("anecdotes_playbook.test", "schedule_config.period"),
			},
		},
	})
}

// A step configuration is optional and computed, so removing it from the
// configuration leaves the stored value in place rather than clearing it.
func TestAccPlaybookResource_removingConfigurationKeepsIt(t *testing.T) {
	title := randomName("pb-keepconfig")

	step := func(payload string) string {
		return fmt.Sprintf(`
resource "anecdotes_playbook" "test" {
  title       = %q
  description = "configuration removal"

  steps = [
    {
      title          = "step"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/tf-acc-keepconfig"
      %s
    },
  ]
}`, title, payload)
	}

	checkStoredPayload := func(s *terraform.State) error {
		id, err := stateAttr(s, "anecdotes_playbook.test", "playbook_id")
		if err != nil {
			return err
		}
		playbook, err := testAccNewClient(t).GetPlaybook(context.Background(), id)
		if err != nil {
			return fmt.Errorf("reading playbook %s: %w", id, err)
		}
		if got := playbook.Steps[0].PayloadConfiguration["source"]; got != "terraform" {
			return fmt.Errorf("the platform no longer holds the payload: %v", playbook.Steps[0].PayloadConfiguration)
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: step(`payload_configuration = jsonencode({ source = "terraform" })`),
				Check:  checkStoredPayload,
			},
			{
				// Dropping the attribute is not a change: the stored value stays.
				Config:   step(``),
				PlanOnly: true,
			},
			{
				Config: step(``),
				Check:  checkStoredPayload,
			},
		},
	})
}
