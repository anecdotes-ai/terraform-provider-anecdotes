// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"time"
)

// FrameworkStatus represents the status of a framework
type FrameworkStatus string

// FrameworkAuditorControlStatus defines which control statuses are visible to auditors
type FrameworkAuditorControlStatus struct {
	ApprovedByAuditor bool `json:"APPROVED_BY_AUDITOR"`
	Gap               bool `json:"GAP"`
	InsufficientData  bool `json:"INSUFFICIENT_DATA"`
	InProgress        bool `json:"IN_PROGRESS"`
	Issue             bool `json:"ISSUE"`
	Monitoring        bool `json:"MONITORING"`
	NotApplicable     bool `json:"NOT_APPLICABLE"`
	NotStarted        bool `json:"NOT_STARTED"`
	ReadyForAudit     bool `json:"READY_FOR_AUDIT"`
	UnderReview       bool `json:"UNDER_REVIEW"`
}

// FrameworkAuditorEvidenceStatus defines which evidence statuses are visible to auditors
type FrameworkAuditorEvidenceStatus struct {
	Auditable bool `json:"AUDITABLE"`
	Gap       bool `json:"GAP"`
	NotSet    bool `json:"NOT_SET"`
}

// FrameworkCurrentAudit represents the current audit configuration
type FrameworkCurrentAudit struct {
	AuditID        string     `json:"audit_id,omitempty"`
	AuditName      string     `json:"audit_name,omitempty"`
	AuditStartDate *time.Time `json:"audit_start_date,omitempty"`
	AuditEndDate   *time.Time `json:"audit_end_date,omitempty"`
}

// Framework represents an Anecdotes compliance framework
type Framework struct {
	// Core identification
	FrameworkID          string          `json:"framework_id"`
	FrameworkName        string          `json:"framework_name"`
	FrameworkDescription string          `json:"framework_description,omitempty"`
	FrameworkStatus      FrameworkStatus `json:"framework_status"`

	// Adoption status
	IsApplicable bool `json:"is_applicable"`

	// Auditor configuration
	FrameworkAuditable                bool                           `json:"framework_auditable"`
	CanAuditorDownloadEvidence        bool                           `json:"can_auditor_download_evidence"`
	CanAuditorViewControlAttachments  bool                           `json:"can_auditor_view_control_attachments"`
	CanAuditorViewControlCustomFields bool                           `json:"can_auditor_view_control_custom_fields"`
	CanAuditorViewSoaReport           bool                           `json:"can_auditor_view_soa_report"`
	CanAuditorViewTags                bool                           `json:"can_auditor_view_tags"`
	FrameworkAuditorControlStatus     FrameworkAuditorControlStatus  `json:"framework_auditor_control_status"`
	FrameworkAuditorEvidenceStatus    FrameworkAuditorEvidenceStatus `json:"framework_auditor_evidence_status"`

	// Structure and references
	FrameworkControlsCategories []string `json:"framework_controls_categories,omitempty"`
	FrameworkReferenceFieldName string   `json:"framework_reference_field_name,omitempty"`
	FrameworkReferences         []string `json:"framework_references,omitempty"`

	// Audit configuration
	FrameworkCurrentAudit *FrameworkCurrentAudit `json:"framework_current_audit,omitempty"`

	// Framework origin and duplication
	FrameworkDuplicatedFrom string `json:"framework_duplicated_from,omitempty"`

	// Plugin/evidence scoping
	FrameworkExcludedPlugins map[string]interface{} `json:"framework_excluded_plugins,omitempty"`

	// Customization
	FrameworkIconID string `json:"framework_icon_id,omitempty"`

	// Ordering
	UnadoptedOrder int `json:"unadopted_order,omitempty"`
}

// ControlStatusObject represents the control status as returned by the API
type ControlStatusObject struct {
	Status       string `json:"status"`
	UpdatedBy    string `json:"updated_by,omitempty"`
	LastEditTime string `json:"last_edit_time,omitempty"`
	Note         string `json:"note,omitempty"`
	ManuallySet  bool   `json:"manually_set,omitempty"`
	GapDetected  bool   `json:"gap_detected,omitempty"`
}

// Control represents a compliance control within a framework
type Control struct {
	// Core identification
	ControlID   string `json:"control_id"`
	FrameworkID string `json:"control_framework_id"`
	ControlName string `json:"control_name"`

	// Control content
	ControlDescription         string `json:"control_description,omitempty"`
	ControlCategory            string `json:"control_category,omitempty"`
	ControlFrameworkCategory   string `json:"control_group,omitempty"`
	ControlFrameworkCategoryID string `json:"control_framework_category_id,omitempty"`

	// Status tracking
	ControlStatus          ControlStatusObject `json:"control_status"`
	ControlStatusUpdatedBy string              `json:"control_status_updated_by,omitempty"`

	// Only custom controls can be modified. A nil value means the platform did
	// not report it.
	ControlIsCustom *bool `json:"control_is_custom,omitempty"`

	// Ownership and assignment
	ControlOwners []string `json:"control_owner,omitempty"`

	// Tagging and classification
	ControlTags              []string `json:"control_tags,omitempty"`
	ControlUnifiedCategories []string `json:"control_unified_categories,omitempty"` // UCV mapping

	// Linked requirements
	LinkedRequirements []LinkedRequirement `json:"linked_requirements,omitempty"`
	RequirementIDs     []string            `json:"control_requirement_ids,omitempty"`

	// Common control mapping (for Unified Control View)
	CommonControlMappings []string `json:"common_control_mappings,omitempty"`

	// Metadata
	LastEditTime string `json:"control_last_edit_time,omitempty"`
}

// LinkedRequirement represents a requirement linked to a control
type LinkedRequirement struct {
	RequirementID   string `json:"requirement_id"`
	RequirementName string `json:"requirement_name,omitempty"`
}

// MaturityLevel represents the valid control maturity levels (matches the
// platform MaturityLevel enum).
type MaturityLevel string

const (
	MaturityLevelInitial    MaturityLevel = "INITIAL"
	MaturityLevelRepeatable MaturityLevel = "REPEATABLE"
	MaturityLevelDefined    MaturityLevel = "DEFINED"
	MaturityLevelManaged    MaturityLevel = "MANAGED"
	MaturityLevelOptimizing MaturityLevel = "OPTIMIZING"
)

// ValidMaturityLevels returns all valid control maturity levels.
func ValidMaturityLevels() []string {
	return []string{
		string(MaturityLevelInitial),
		string(MaturityLevelRepeatable),
		string(MaturityLevelDefined),
		string(MaturityLevelManaged),
		string(MaturityLevelOptimizing),
	}
}

// ValidControlStatuses returns the valid control status values (platform
// ControlStatus enum, UPPER_SNAKE) accepted by the controls data-source
// status filter.
// NOTE: near-twin of ValidAuditorControlStatuses (10 lower_snake values, no
// NOT_READY_FOR_AUDIT) — the sets and casings differ on purpose; never merge.
func ValidControlStatuses() []string {
	return []string{
		"NOT_STARTED",
		"IN_PROGRESS",
		"READY_FOR_AUDIT",
		"GAP",
		"ISSUE",
		"APPROVED_BY_AUDITOR",
		"MONITORING",
		"NOT_APPLICABLE",
		"NOT_READY_FOR_AUDIT",
		"INSUFFICIENT_DATA",
		"UNDER_REVIEW",
	}
}

// ValidAuditorControlStatuses returns the valid member values for the
// framework auditor_visible_control_statuses set attribute. These correspond
// to the boolean sub-fields of the FrameworkAuditorControlStatus API object.
// NOTE: near-twin of ValidControlStatuses (11 UPPER_SNAKE platform statuses) —
// the sets and casings differ on purpose; never merge.
func ValidAuditorControlStatuses() []string {
	return []string{
		"approved_by_auditor",
		"gap",
		"in_progress",
		"insufficient_data",
		"issue",
		"monitoring",
		"not_applicable",
		"not_started",
		"ready_for_audit",
		"under_review",
	}
}

// ValidAuditorEvidenceStatuses returns the valid member values for the
// framework auditor_visible_evidence_statuses set attribute. These correspond
// to the boolean sub-fields of the FrameworkAuditorEvidenceStatus API object.
func ValidAuditorEvidenceStatuses() []string {
	return []string{
		"auditable",
		"gap",
		"not_set",
	}
}

// ValidRequirementCategories returns the requirement categories Anecdotes
// defines. New requirements must use one of these values.
func ValidRequirementCategories() []string {
	return []string{
		"AI Specific",
		"Access",
		"Backup",
		"CIS AWS Benchmarks specific",
		"Cardholder Data Environment",
		"Custom Requirements",
		"Documentation",
		"Encryption",
		"FedRAMP specific",
		"Human Resources",
		"IRAP specific",
		"ISMS",
		"IT & Network",
		"Incident Response & Support",
		"Infrastructure",
		"Logging & Monitoring",
		"Organization",
		"Physical",
		"Privacy",
		"Processing Integrity",
		"SDLC & Change management",
		"SSPA specific",
		"Security",
		"Training",
		"Vendors & Customers",
		"Vulnerabilities",
	}
}

// Requirement represents a compliance requirement in the Requirements Hub
type Requirement struct {
	RequirementID            string   `json:"requirement_id"`
	RequirementName          string   `json:"requirement_name"`
	RequirementDescription   string   `json:"requirement_description,omitempty"`
	RequirementHelp          string   `json:"requirement_help,omitempty"`
	RequirementCategory      string   `json:"requirement_category,omitempty"`
	RequirementStatus        string   `json:"requirement_status,omitempty"`
	RequirementStatusName    string   `json:"requirement_status_name,omitempty"`
	RequirementStatusID      string   `json:"requirement_status_id,omitempty"` // the API returns the status as an ID; name is resolved client-side
	RequirementOwners        []string `json:"requirement_owners,omitempty"`
	RequirementEditedBy      string   `json:"requirement_edited_by,omitempty"`
	RequirementLastEditTime  string   `json:"requirement_last_edit_time,omitempty"`
	RequirementIsCustom      bool     `json:"requirement_is_custom,omitempty"`
	RequirementApplicability bool     `json:"requirement_applicability,omitempty"`
	RequirementNoteExists    bool     `json:"requirement_note_exists,omitempty"`

	// Requirement View fields. RequirementParentID is set (and immutable) only for
	// a view; ViewName is its display name — requirement_name resolves to the
	// parent's name for a view, not the view's own name, so callers must read
	// ViewName directly rather than RequirementName.
	RequirementParentID string `json:"requirement_parent_id,omitempty"`
	ViewName            string `json:"view_name,omitempty"`

	// Linked entities (from API)
	RequirementRelatedControls   []string               `json:"requirement_related_controls,omitempty"`
	RequirementRelatedFrameworks []string               `json:"requirement_related_frameworks,omitempty"`
	RequirementEvidenceIDs       []string               `json:"requirement_evidence_ids,omitempty"`
	RequirementRelatedPolicies   []string               `json:"requirement_related_policies_ids,omitempty"`
	RequirementScopingOverrides  map[string]interface{} `json:"requirement_scoping_overrides,omitempty"`

	// Services that automate this requirement
	ServicesThatAutomate []string `json:"services_that_automates,omitempty"`
}

// ControlRequirementLink represents a link between a control and requirement
type ControlRequirementLink struct {
	ControlID     string `json:"control_id"`
	RequirementID string `json:"requirement_id"`
	FrameworkID   string `json:"framework_id"`
}

// FrameworkCreateRequest is the create-framework body. The create endpoint
// ignores auditor configuration; it is applied afterward via UpdateFramework and
// SetFrameworkAuditorControlStatus / SetFrameworkAuditorEvidenceStatus.
type FrameworkCreateRequest struct {
	FrameworkName        string `json:"framework_name"`
	FrameworkDescription string `json:"framework_description"`
	FolderID             string `json:"folder_id,omitempty"`
}

// FrameworkUpdateRequest represents the request body for updating a framework
type FrameworkUpdateRequest struct {
	FrameworkName        string `json:"framework_name,omitempty"`
	FrameworkDescription string `json:"framework_description"`

	// The visibility status sets use dedicated endpoints, not this body.
	// framework_auditable is platform-managed and cannot be set here.
	CanAuditorDownloadEvidence        *bool `json:"can_auditor_download_evidence,omitempty"`
	CanAuditorViewControlAttachments  *bool `json:"can_auditor_view_control_attachments,omitempty"`
	CanAuditorViewControlCustomFields *bool `json:"can_auditor_view_control_custom_fields,omitempty"`
	CanAuditorViewSoaReport           *bool `json:"can_auditor_view_soa_report,omitempty"`
	CanAuditorViewTags                *bool `json:"can_auditor_view_tags,omitempty"`
}

// ControlCategory represents a control category within a framework
type ControlCategory struct {
	CategoryID   string `json:"category_id,omitempty"`
	CategoryName string `json:"category_name"`
	FrameworkID  string `json:"framework_id"`
	LastEditTime string `json:"last_edit_time,omitempty"`
}

// ControlCategoryCreateRequest represents the request body for creating a category
type ControlCategoryCreateRequest struct {
	CategoryName string `json:"category_name"`
	FrameworkID  string `json:"framework_id"`
}

// ControlCategoryUpdateRequest represents the request body for updating a category
type ControlCategoryUpdateRequest struct {
	CategoryName string `json:"category_name"`
}

// ControlCreateRequest represents the request body for creating a control
type ControlCreateRequest struct {
	ControlName                string            `json:"control_name"`
	ControlDescription         string            `json:"control_description,omitempty"`
	ControlFrameworkCategory   string            `json:"control_framework_category"`
	ControlFrameworkCategoryID string            `json:"control_framework_category_id,omitempty"`
	ControlOwners              []string          `json:"control_owner,omitempty"`
	ControlTags                []string          `json:"control_tags,omitempty"`
	CommonControlMappings      []string          `json:"common_control_mappings,omitempty"`
	CustomTextFields           map[string]string `json:"custom_text_fields,omitempty"`
	CustomDropdownFields       map[string]string `json:"custom_dropdown_fields,omitempty"`
	CustomTagsGroups           map[string]string `json:"custom_tags_groups,omitempty"`
}

// ControlUpdateRequest represents the request body for updating a control
type ControlUpdateRequest struct {
	ControlName                string            `json:"control_name,omitempty"`
	ControlDescription         string            `json:"control_description"`
	ControlFrameworkCategory   string            `json:"control_framework_category,omitempty"`
	ControlFrameworkCategoryID string            `json:"control_framework_category_id,omitempty"`
	ControlTags                []string          `json:"control_tags,omitempty"`
	CommonControlMappings      []string          `json:"common_control_mappings,omitempty"`
	CustomTextFields           map[string]string `json:"custom_text_fields,omitempty"`
	CustomDropdownFields       map[string]string `json:"custom_dropdown_fields,omitempty"`
	CustomTagsGroups           map[string]string `json:"custom_tags_groups,omitempty"`
}

// RequirementCreateRequest represents the request body for creating a requirement
type RequirementCreateRequest struct {
	RequirementDescription       string   `json:"requirement_description"`                // The name/title (required)
	RequirementHelp              string   `json:"requirement_help,omitempty"`             // Detailed description
	RequirementCategory          string   `json:"requirement_category,omitempty"`         // Category
	RequirementOwners            []string `json:"requirement_owners,omitempty"`           // Owner emails
	RequirementRelatedControls   []string `json:"requirement_related_controls,omitempty"` // Linked control IDs
	RequirementRelatedFrameworks []string `json:"requirement_related_frameworks"`         // Linked framework IDs (required, can be empty)
}

// RequirementUpdateRequest represents the request body for updating a requirement.
// NOTE: The API expects this wrapped in {"requirement": {...}} — see UpdateRequirement in client.go.
type RequirementUpdateRequest struct {
	RequirementDescription string `json:"requirement_description,omitempty"`
	// Pointers distinguish "leave unchanged" (nil) from "replace with this
	// value", so a partial update does not overwrite fields it did not set.
	RequirementHelp     *string `json:"requirement_help,omitempty"`
	RequirementCategory string  `json:"requirement_category,omitempty"`

	RequirementOwners            *[]string              `json:"requirement_owners,omitempty"`
	RequirementRelatedControls   []string               `json:"requirement_related_controls,omitempty"`
	RequirementRelatedFrameworks []string               `json:"requirement_related_frameworks,omitempty"`
	RequirementRelatedEvidences  *[]string              `json:"requirement_related_evidences,omitempty"`
	EvidenceIDs                  []string               `json:"evidence_ids,omitempty"`
	RequirementScopingOverrides  map[string]interface{} `json:"requirement_scoping_overrides,omitempty"`
}

// RequirementViewCreateRequest represents the request body for creating a
// Requirement View: a requirement scoped beneath a parent requirement, using
// view_name in place of requirement_description.
//
// Unlike RequirementCreateRequest, this has no related-controls/frameworks
// fields: on the create endpoint those only drive a one-time server-side
// swap and are never persisted on the view itself (confirmed against the
// live API — the create/update response never echoes them back), so there
// is nothing for Terraform to own or reconcile here. Control associations
// for a view are managed the same way as for any other requirement, through
// anecdotes_mapping_control_requirement.
type RequirementViewCreateRequest struct {
	RequirementParentID string   `json:"requirement_parent_id"` // The parent requirement this view is scoped beneath (required)
	ViewName            string   `json:"view_name"`             // The view's display name (required)
	RequirementCategory string   `json:"requirement_category,omitempty"`
	RequirementOwners   []string `json:"requirement_owners,omitempty"`
}

// RequirementViewUpdateRequest represents the request body for updating a
// Requirement View. requirement_parent_id is immutable and is never sent here.
// NOTE: The API expects this wrapped in {"requirement": {...}} — see UpdateRequirementView in client.go.
type RequirementViewUpdateRequest struct {
	ViewName            *string `json:"view_name,omitempty"`
	RequirementCategory string  `json:"requirement_category,omitempty"`

	RequirementOwners *[]string `json:"requirement_owners,omitempty"`
}

// Folder represents a folder for organizing frameworks
type Folder struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	FrameworksList []string `json:"frameworks_list,omitempty"`
}

// FolderCreateRequest represents the request body for creating a folder
type FolderCreateRequest struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	FrameworksList []string `json:"frameworks_list"`
}

// FolderUpdateRequest represents the request body for updating a folder
type FolderUpdateRequest struct {
	Name           string   `json:"name,omitempty"`
	FrameworksList []string `json:"frameworks_list,omitempty"`
}

// EvidenceType represents the type of an evidence item. The API uppercases
// input server-side, but the provider validates the canonical uppercase form.
type EvidenceType string

const (
	EvidenceTypeManual  EvidenceType = "MANUAL"
	EvidenceTypeURL     EvidenceType = "URL"
	EvidenceTypeLink    EvidenceType = "LINK"
	EvidenceTypeList    EvidenceType = "LIST"
	EvidenceTypeTicket  EvidenceType = "TICKET"
	EvidenceTypeBuilder EvidenceType = "BUILDER"
	EvidenceTypeAPI     EvidenceType = "API"
	EvidenceTypeMerged  EvidenceType = "MERGED"
)

// ValidEvidenceTypes returns all valid evidence type values (canonical uppercase).
func ValidEvidenceTypes() []string {
	return []string{
		string(EvidenceTypeManual),
		string(EvidenceTypeURL),
		string(EvidenceTypeLink),
		string(EvidenceTypeList),
		string(EvidenceTypeTicket),
		string(EvidenceTypeBuilder),
		string(EvidenceTypeAPI),
		string(EvidenceTypeMerged),
	}
}

// Evidence represents an evidence item in the Anecdotes Evidence Pool.
// Evidence can be automatically collected via plugins, manually uploaded, or created as URL references.
type Evidence struct {
	// Core identity
	EvidenceID         string `json:"evidence_id"`
	EvidenceInstanceID string `json:"evidence_instance_id"`
	EvidenceName       string `json:"evidence_name"`
	EvidenceType       string `json:"evidence_type"`

	// Display & content
	EvidenceDisplayName *string `json:"evidence_display_name"`
	EvidenceURL         *string `json:"evidence_url"`
	EvidenceSummary     *string `json:"evidence_summary"`

	// Service/plugin info
	EvidenceServiceID          string `json:"evidence_service_id"`
	EvidenceServiceDisplayName string `json:"evidence_service_display_name"`

	// Flags
	EvidenceIsApplicable bool `json:"evidence_is_applicable"`
	EvidenceIsCustom     bool `json:"evidence_is_custom"`
	EvidenceIsBeta       bool `json:"evidence_is_beta"`
	EvidenceIsUAR        bool `json:"evidence_is_uar"`
	EvidenceIsSOT        bool `json:"evidence_is_sot"`
	EvidenceIsSeen       bool `json:"evidence_is_seen"`

	// Timestamps
	EvidenceCollectionTimestamp string  `json:"evidence_collection_timestamp"`
	EvidenceLastModifiedTime    *string `json:"evidence_last_modified_time"`
	EvidenceCreationTime        *string `json:"evidence_creation_time"`
	FreshnessTimestamp          *string `json:"freshness_timestamp"`

	// Users
	EvidenceUploadedBy     string  `json:"evidence_uploaded_by"`
	EvidenceLastModifiedBy *string `json:"evidence_last_modified_by"`

	// Analysis
	EvidenceAlertLevel             int         `json:"evidence_alert_level"`
	EvidenceAlertLevelGapRulesOnly int         `json:"evidence_alert_level_gap_rules_only"`
	EvidenceItemsCount             int         `json:"evidence_items_count"`
	EvidenceEntityType             string      `json:"evidence_entity_type"`
	EvidenceParentID               string      `json:"evidence_parent_id"`
	EvidenceProcessingState        *string     `json:"evidence_processing_state"`
	EvidenceGapMessage             *string     `json:"evidence_gap_message"`
	EvidenceGap                    interface{} `json:"evidence_gap"`
}

// EvidenceCreateResponse represents the response from POST /evidence/v1/evidence/manual.
// The API returns an array of created evidence IDs (supports batch creation).
type EvidenceCreateResponse struct {
	EvidenceID []string `json:"evidence_id"`
}

// AttachmentUploadRequest represents the request body for getting a presigned upload URL.
type AttachmentUploadRequest struct {
	Context AttachmentContext `json:"context"`
	File    AttachmentFile    `json:"file"`
}

type AttachmentContext struct {
	Type string `json:"type"`
}

type AttachmentFile struct {
	Filename    string `json:"filename"`
	FileSize    int64  `json:"file_size"`
	ContentType string `json:"content_type"`
}

// Playbook represents an automation playbook and its steps.
type Playbook struct {
	PlaybookID          string                  `json:"playbook_id"`
	PlaybookTitle       string                  `json:"playbook_title"`
	PlaybookDescription string                  `json:"playbook_description"`
	Active              bool                    `json:"active"`
	Type                string                  `json:"type,omitempty"`
	Status              string                  `json:"status,omitempty"`
	RestrictedFeatures  []string                `json:"restricted_features,omitempty"`
	CreatedBy           string                  `json:"created_by,omitempty"`
	CreationTimestamp   string                  `json:"creation_timestamp,omitempty"`
	LastUpdatedBy       string                  `json:"last_updated_by,omitempty"`
	LastUpdateTimestamp string                  `json:"last_update_timestamp,omitempty"`
	ScheduleConfig      *PlaybookScheduleConfig `json:"schedule_config,omitempty"`
	Steps               []PlaybookStep          `json:"steps,omitempty"`
}

// PlaybookScheduleConfig is the recurring-execution schedule of a playbook.
// EndsIn and EndDate are mutually exclusive; EndDate is derived from StartDate
// and EndsIn when only EndsIn is supplied.
type PlaybookScheduleConfig struct {
	Period    string  `json:"period"`
	Time      string  `json:"time"`
	Timezone  string  `json:"timezone,omitempty"`
	StartDate string  `json:"start_date"`
	EndsIn    *string `json:"ends_in,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
}

// PlaybookStep is a single step of a playbook.
type PlaybookStep struct {
	StepID               string                 `json:"step_id"`
	StepTitle            string                 `json:"step_title"`
	PlaybookID           string                 `json:"playbook_id,omitempty"`
	StepTriggerEvent     string                 `json:"step_trigger_event"`
	StepActionType       string                 `json:"step_action_type"`
	InternalAction       bool                   `json:"internal_action"`
	StepURLToTrigger     string                 `json:"step_url_to_trigger,omitempty"`
	FilterConfiguration  map[string]interface{} `json:"filter_configuration,omitempty"`
	PayloadConfiguration map[string]interface{} `json:"payload_configuration,omitempty"`
	HeadersConfiguration map[string]interface{} `json:"headers_configuration,omitempty"`
	LastRunTimestamp     *string                `json:"last_run_timestamp,omitempty"`
	LastRunStatus        string                 `json:"last_run_status,omitempty"`
}

// PlaybookCreateRequest represents the request body for creating a playbook
// together with all of its steps.
type PlaybookCreateRequest struct {
	PlaybookTitle       string                  `json:"playbook_title"`
	PlaybookDescription string                  `json:"playbook_description"`
	Steps               []PlaybookStepInput     `json:"steps"`
	ScheduleConfig      *PlaybookScheduleConfig `json:"schedule_config,omitempty"`
}

// PlaybookStepInput is a step as supplied on create. InternalAction and the
// placeholder trigger URL of an internal step are set by the platform.
type PlaybookStepInput struct {
	StepID               string                 `json:"step_id,omitempty"`
	StepTitle            string                 `json:"step_title"`
	StepTriggerEvent     string                 `json:"step_trigger_event"`
	StepActionType       string                 `json:"step_action_type"`
	StepURLToTrigger     string                 `json:"step_url_to_trigger,omitempty"`
	FilterConfiguration  map[string]interface{} `json:"filter_configuration,omitempty"`
	PayloadConfiguration map[string]interface{} `json:"payload_configuration,omitempty"`
	HeadersConfiguration map[string]interface{} `json:"headers_configuration,omitempty"`
}

// PlaybookUpdateRequest represents the request body for updating a playbook.
// Only the fields set here are changed; steps listed in Steps must already
// exist on the playbook.
type PlaybookUpdateRequest struct {
	PlaybookTitle       *string                 `json:"playbook_title,omitempty"`
	PlaybookDescription *string                 `json:"playbook_description,omitempty"`
	Active              *bool                   `json:"active,omitempty"`
	Steps               []PlaybookStepUpdate    `json:"steps,omitempty"`
	ScheduleConfig      *PlaybookScheduleConfig `json:"schedule_config,omitempty"`
}

// PlaybookStepUpdate is a step as supplied on update. A nil configuration is
// left unchanged; an empty one clears it.
type PlaybookStepUpdate struct {
	StepID               string                  `json:"step_id"`
	StepTitle            *string                 `json:"step_title,omitempty"`
	StepTriggerEvent     *string                 `json:"step_trigger_event,omitempty"`
	StepActionType       *string                 `json:"step_action_type,omitempty"`
	StepURLToTrigger     *string                 `json:"step_url_to_trigger,omitempty"`
	InternalAction       *bool                   `json:"internal_action,omitempty"`
	FilterConfiguration  *map[string]interface{} `json:"filter_configuration,omitempty"`
	PayloadConfiguration *map[string]interface{} `json:"payload_configuration,omitempty"`
	HeadersConfiguration *map[string]interface{} `json:"headers_configuration,omitempty"`
}

// PlaybookLibraryEvent is a trigger event a playbook step can subscribe to.
// A step subscribes using TriggerKey when it is set, and EventType otherwise.
type PlaybookLibraryEvent struct {
	EventType            string               `json:"event_type"`
	TriggerKey           string               `json:"trigger_key,omitempty"`
	EventText            string               `json:"event_text"`
	Category             string               `json:"category"`
	Description          string               `json:"description,omitempty"`
	Icon                 string               `json:"icon,omitempty"`
	IsAvailable          bool                 `json:"is_available"`
	SupportedActions     []string             `json:"supported_actions,omitempty"`
	ComingSoonActions    []string             `json:"coming_soon_actions,omitempty"`
	RequiredChangedField string               `json:"required_changed_field,omitempty"`
	EventFields          []PlaybookEventField `json:"event_fields,omitempty"`
}

// PlaybookEventField is a field an event carries. A filterable one can be used
// as the left side of a step's filter_configuration, paired with its operator.
type PlaybookEventField struct {
	FieldID          string   `json:"field_id"`
	DisplayName      string   `json:"display_name,omitempty"`
	Type             string   `json:"type,omitempty"`
	IsFilterable     bool     `json:"is_filterable"`
	AQLOperator      string   `json:"aql_operator,omitempty"`
	FieldDescription string   `json:"field_description,omitempty"`
	Values           []string `json:"values,omitempty"`
}

// PlaybookLibraryAction is an action a playbook step can perform.
type PlaybookLibraryAction struct {
	ActionType     string                `json:"action_type"`
	ActionText     string                `json:"action_text"`
	ActionName     string                `json:"action_name,omitempty"`
	ActionCategory string                `json:"action_category,omitempty"`
	Description    string                `json:"description,omitempty"`
	Icon           string                `json:"icon,omitempty"`
	ComingSoon     bool                  `json:"coming_soon"`
	ActionFields   []PlaybookActionField `json:"action_fields,omitempty"`
}

// PlaybookActionField is a field an action takes. A required one must appear in
// the payload_configuration of a step performing that action.
type PlaybookActionField struct {
	FieldID          string   `json:"field_id"`
	DisplayName      string   `json:"display_name,omitempty"`
	Type             string   `json:"type,omitempty"`
	IsRequired       bool     `json:"is_required"`
	FieldDescription string   `json:"field_description,omitempty"`
	Values           []string `json:"values,omitempty"`
}

// ValidPlaybookStepActionTypes returns the actions a playbook step can perform.
func ValidPlaybookStepActionTypes() []string {
	return []string{
		"assign_control",
		"assign_requirement",
		"audit_sync_comment",
		"audit_sync_control",
		"audit_sync_evidence",
		"call_forge",
		"change_control_status",
		"comment_specific_control",
		"comment_triggered_control",
		"comment_triggered_risk",
		"create_comment",
		"create_finding",
		"create_task",
		"evidence_to_bucket",
		"in_app_notification",
		"notify_via_service",
		"send_email",
		"trigger_plugin_run",
		"update_risk_level",
		"webhook",
	}
}

// ValidPlaybookSchedulePeriods returns the recurrence periods of a scheduled playbook.
func ValidPlaybookSchedulePeriods() []string {
	return []string{"day", "week", "month", "quarter", "year"}
}

// ValidPlaybookScheduleEndsIn returns the durations after which a schedule expires.
func ValidPlaybookScheduleEndsIn() []string {
	return []string{"week", "month", "year"}
}

// ScheduledPlaybookTrigger is the trigger event of a scheduled playbook's first step.
const ScheduledPlaybookTrigger = "ScheduledPlaybookTriggered"
