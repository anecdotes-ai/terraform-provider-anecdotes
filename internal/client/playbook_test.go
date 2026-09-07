// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// playbookServer serves the token exchange, records the body of the first
// write it receives, and answers every playbook read with the given list.
func playbookServer(t *testing.T, list string) (*httptest.Server, *[]byte) {
	t.Helper()

	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/apikey/exchange"):
			_, _ = w.Write([]byte("test-token"))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(list))
		default:
			if captured == nil {
				captured, _ = io.ReadAll(r.Body)
			}
			_, _ = w.Write([]byte(`{"playbook_id":"pb1"}`))
		}
	}))
	t.Cleanup(srv.Close)

	return srv, &captured
}

const onePlaybook = `[{"playbook_id":"pb1","playbook_title":"t","playbook_description":"d","active":true,
	"type":"playbook","status":"published","schedule_config":null,
	"steps":[{"step_id":"s1","step_title":"one","step_trigger_event":"ControlStatusChanged",
	"step_action_type":"webhook","internal_action":false,
	"step_url_to_trigger":"https://example.com/hook",
	"filter_configuration":{},"payload_configuration":{"a":1},"headers_configuration":{}}]}]`

func TestCreatePlaybook_SendsStepsAndObjectConfigurations(t *testing.T) {
	srv, captured := playbookServer(t, onePlaybook)

	_, err := newTestClient(t, srv).CreatePlaybook(context.Background(), &PlaybookCreateRequest{
		PlaybookTitle:       "t",
		PlaybookDescription: "d",
		Steps: []PlaybookStepInput{{
			StepID:              "s1",
			StepTitle:           "one",
			StepTriggerEvent:    "ControlStatusChanged",
			StepActionType:      "webhook",
			StepURLToTrigger:    "https://example.com/hook",
			FilterConfiguration: map[string]interface{}{"left": "control_status"},
		}},
	})
	if err != nil {
		t.Fatalf("CreatePlaybook: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(*captured, &payload); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, *captured)
	}

	steps, ok := payload["steps"].([]interface{})
	if !ok || len(steps) != 1 {
		t.Fatalf("steps must be sent as a list of one, got keys %v", keysOf(payload))
	}
	step := steps[0].(map[string]interface{})

	// A configuration must travel as a JSON object. Encoded as a string it is
	// rejected.
	if _, ok := step["filter_configuration"].(map[string]interface{}); !ok {
		t.Errorf("filter_configuration must be an object, got %T", step["filter_configuration"])
	}
	// internal_action is derived by the platform from the trigger URL.
	if _, present := step["internal_action"]; present {
		t.Error("internal_action must not be sent on create")
	}
	for _, field := range []string{"step_id", "step_title", "step_trigger_event", "step_action_type"} {
		if _, present := step[field]; !present {
			t.Errorf("%s must be sent, got keys %v", field, keysOf(step))
		}
	}
}

func TestUpdatePlaybook_DistinguishesUnsetFromCleared(t *testing.T) {
	srv, captured := playbookServer(t, onePlaybook)

	empty := map[string]interface{}{}
	inactive := false
	_, err := newTestClient(t, srv).UpdatePlaybook(context.Background(), "pb1", &PlaybookUpdateRequest{
		Active: &inactive,
		Steps: []PlaybookStepUpdate{{
			StepID:               "s1",
			PayloadConfiguration: &empty,
		}},
	})
	if err != nil {
		t.Fatalf("UpdatePlaybook: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(*captured, &payload); err != nil {
		t.Fatalf("unmarshal: %v (body %s)", err, *captured)
	}

	// active=false must survive: dropping it leaves the playbook enabled.
	if active, present := payload["active"]; !present || active != false {
		t.Errorf("active=false must be sent, got %v (keys %v)", active, keysOf(payload))
	}

	step := payload["steps"].([]interface{})[0].(map[string]interface{})
	// An empty configuration clears it; an absent one leaves it untouched.
	got, present := step["payload_configuration"]
	if !present {
		t.Errorf("an empty payload_configuration must be sent to clear it, got keys %v", keysOf(step))
	} else if m, ok := got.(map[string]interface{}); !ok || len(m) != 0 {
		t.Errorf("payload_configuration must be sent as an empty object, got %v", got)
	}
	if _, present := step["step_title"]; present {
		t.Error("an unset step field must not be sent")
	}
}

func TestGetPlaybook_NotFound(t *testing.T) {
	srv, _ := playbookServer(t, onePlaybook)

	_, err := newTestClient(t, srv).GetPlaybook(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListPlaybooks_ParsesSteps(t *testing.T) {
	srv, _ := playbookServer(t, onePlaybook)

	playbooks, err := newTestClient(t, srv).ListPlaybooks(context.Background())
	if err != nil {
		t.Fatalf("ListPlaybooks: %v", err)
	}
	if len(playbooks) != 1 || len(playbooks[0].Steps) != 1 {
		t.Fatalf("expected one playbook with one step, got %+v", playbooks)
	}
	step := playbooks[0].Steps[0]
	if step.StepID != "s1" || step.StepActionType != "webhook" {
		t.Errorf("unexpected step: %+v", step)
	}
	if got := step.PayloadConfiguration["a"]; got != float64(1) {
		t.Errorf("payload_configuration must decode into a map, got %v", got)
	}
}
