# A playbook that posts to an external endpoint whenever a control's status changes.
resource "anecdotes_playbook" "control_status_webhook" {
  title       = "Notify on control status change"
  description = "Posts to the compliance webhook whenever a control changes status"

  steps = [
    {
      title          = "Post to the compliance webhook"
      trigger_event  = "ControlStatusChanged"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/hooks/anecdotes"

      # Field names come from the event's own fields, which the
      # anecdotes_playbook_library data source reports.
      payload_configuration = jsonencode({
        control = "{{ extra_payload.control_name }}"
        status  = "{{ extra_payload.new }}"
      })

      # Only fire for controls that moved into IN_PROGRESS. left is a
      # filterable field of the event; operator is that field's aql_operator.
      filter_configuration = jsonencode({
        left     = "extra_payload.new"
        operator = "IsIn"
        right    = ["IN_PROGRESS"]
      })
      headers_configuration = jsonencode({
        "X-Source" = "anecdotes"
      })
    },
  ]
}

# Two steps, chained: the second runs after the first. Chaining points a step's
# trigger_event at the step_id of the step it follows, so those ids are set
# explicitly rather than generated.
#
# An action must be one the trigger supports. Each event in the
# anecdotes_playbook_library data source reports its own supported_actions, and
# an action outside that set is stored but does not run.
resource "anecdotes_playbook" "gap_escalation" {
  title       = "Escalate evidence gaps"
  description = "Opens a finding on an evidence gap, then posts to a webhook"

  steps = [
    {
      step_id       = "6f9619ff-8b86-4011-b42d-00c04fc964ff"
      title         = "Open a finding"
      trigger_event = "EvidenceGapDetected"
      action_type   = "create_finding"

      # Each action has its own required fields, carried in the payload. The
      # anecdotes_playbook_action_library data source reports which are required.
      payload_configuration = jsonencode({
        title       = "Evidence gap detected"
        severity    = "MEDIUM"
        reported_by = "compliance@example.com"
      })
    },
    {
      title          = "Notify the compliance webhook"
      trigger_event  = "6f9619ff-8b86-4011-b42d-00c04fc964ff"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/hooks/evidence-gap"
    },
  ]
}

# A playbook that runs on a schedule. The first step's trigger_event must be
# ScheduledPlaybookTriggered, and the platform owns that step's filter.
resource "anecdotes_playbook" "weekly_digest" {
  title       = "Weekly compliance digest"
  description = "Sends a digest of the week's compliance activity"

  schedule_config = {
    period     = "week"
    time       = "09:30"
    timezone   = "America/New_York"
    start_date = "2026-09-07T00:00:00Z"
    ends_in    = "year"
  }

  steps = [
    {
      title          = "Send the digest"
      trigger_event  = "ScheduledPlaybookTriggered"
      action_type    = "webhook"
      url_to_trigger = "https://example.com/hooks/digest"
    },
  ]
}
