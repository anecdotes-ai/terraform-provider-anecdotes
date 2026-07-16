// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// ErrorClass categorizes an API failure so the provider layer can choose the
// right Terraform behavior (retry, drop-from-state, friendly auth message, …)
// and an appropriate diagnostic summary.
type ErrorClass string

const (
	ClassValidation  ErrorClass = "validation"       // 400/422 — bad input
	ClassPermission  ErrorClass = "permission"       // 401/403 — auth/permission
	ClassFeatureGate ErrorClass = "feature_disabled" // 402 — tenant feature not enabled
	ClassNotFound    ErrorClass = "not_found"        // 404 — resource missing
	ClassConflict    ErrorClass = "conflict"         // 409 — already exists / conflicting
	ClassUnsupported ErrorClass = "unsupported"      // 405 — operation not supported
	ClassServer      ErrorClass = "server"           // 5xx — server-side error
	ClassUnknown     ErrorClass = "unknown"          // anything else
)

// FieldError is a single attribute-scoped validation error decoded from a
// Pydantic-style error envelope.
type FieldError struct {
	// Attribute is the field name the error applies to, with the framework
	// location prefix (body/query/path) stripped — e.g. "name".
	Attribute string
	// Detail is the human-readable reason, e.g. "field required".
	Detail string
}

// APIError is a structured, decoded representation of a non-2xx API response.
// Every exported field is safe to show to a user: the raw response body,
// request IDs, HTML error pages, and credentials are never stored here.
type APIError struct {
	StatusCode int
	Method     string
	Path       string
	Class      ErrorClass
	// Message is a clean, user-facing description of the failure.
	Message string
	// FieldErrors is populated for validation failures (Class == ClassValidation).
	FieldErrors []FieldError
}

// Error implements the error interface and returns the clean, redacted message.
// This is what gets printed by existing `fmt.Sprintf("...: %s", err)` call sites.
func (e *APIError) Error() string { return e.Message }

// IsValidation reports whether the failure is a client-side input/validation error.
func (e *APIError) IsValidation() bool { return e.Class == ClassValidation }

// IsPermission reports whether the failure is an auth/permission error.
func (e *APIError) IsPermission() bool { return e.Class == ClassPermission }

// IsNotFound reports whether the referenced resource does not exist.
func (e *APIError) IsNotFound() bool { return e.Class == ClassNotFound }

// IsConflict reports whether the failure is a conflict (already exists, etc.).
func (e *APIError) IsConflict() bool { return e.Class == ClassConflict }

// IsFeatureGated reports whether the operation requires a tenant feature that is
// not enabled (HTTP 402).
func (e *APIError) IsFeatureGated() bool { return e.Class == ClassFeatureGate }

// IsUnsupported reports whether the operation is not supported by the API (405).
func (e *APIError) IsUnsupported() bool { return e.Class == ClassUnsupported }

// IsRetryable reports whether retrying the request might succeed. doRequest
// already exhausts these internally, so a surfaced APIError is effectively final.
func (e *APIError) IsRetryable() bool { return isRetryable(e.StatusCode) }

// classifyStatus maps an HTTP status code to an ErrorClass.
func classifyStatus(status int) ErrorClass {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity: // 400, 422
		return ClassValidation
	case http.StatusUnauthorized, http.StatusForbidden: // 401, 403
		return ClassPermission
	case http.StatusPaymentRequired: // 402
		return ClassFeatureGate
	case http.StatusNotFound: // 404
		return ClassNotFound
	case http.StatusMethodNotAllowed: // 405
		return ClassUnsupported
	case http.StatusConflict: // 409
		return ClassConflict
	}
	if status >= 500 {
		return ClassServer
	}
	return ClassUnknown
}

// genericMessage returns a safe, user-facing message for a status code when the
// response body is unusable (empty, HTML, internal, or unrecognized). It never
// echoes the raw body.
func genericMessage(status int, class ErrorClass) string {
	switch class {
	case ClassPermission:
		return "The Anecdotes API rejected the credentials (HTTP 401/403). The API token may have expired or lacks permission for this operation. Verify ANECDOTES_API_KEY."
	case ClassServer:
		return fmt.Sprintf("The Anecdotes API encountered an internal error (HTTP %d). This is a server-side issue — please retry; if it persists, contact Anecdotes support.", status)
	case ClassNotFound:
		return "The requested resource was not found (HTTP 404)."
	case ClassUnsupported:
		return "This operation is not supported by the Anecdotes API (HTTP 405)."
	case ClassFeatureGate:
		return "This operation requires an Anecdotes feature that is not enabled for your tenant (HTTP 402)."
	default:
		return fmt.Sprintf("The Anecdotes API request failed with HTTP %d.", status)
	}
}

// looksLikeHTML reports whether a body is an HTML page (e.g. a Flask 500 page)
// rather than a JSON envelope.
func looksLikeHTML(body string) bool {
	t := strings.TrimSpace(strings.ToLower(body))
	return strings.HasPrefix(t, "<!doctype") || strings.HasPrefix(t, "<html")
}

// looksInternal reports whether a decoded message exposes server internals that
// should not be surfaced to users (stack traces, task-group dumps, etc.).
func looksInternal(msg string) bool {
	l := strings.ToLower(msg)
	for _, marker := range []string{"traceback", "unhandled errors in a taskgroup", "internal server error", "facade for"} {
		if strings.Contains(l, marker) {
			return true
		}
	}
	return false
}

// stripLocPrefix removes the leading framework location segment (body/query/path)
// from a Pydantic `loc` array and returns a dotted attribute path.
func stripLocPrefix(loc []interface{}) string {
	parts := make([]string, 0, len(loc))
	for i, seg := range loc {
		s := fmt.Sprintf("%v", seg)
		if i == 0 && (s == "body" || s == "query" || s == "path") {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ".")
}

// pydanticDetail models both Pydantic v1 and v2 array-style validation entries.
type pydanticDetail struct {
	Loc []interface{} `json:"loc"`
	Msg string        `json:"msg"`
	Typ string        `json:"type"`
}

// parseAPIError decodes a non-2xx response into a structured, redacted APIError.
// It recognizes the seven envelope shapes the Anecdotes services emit and falls
// back to a safe generic message for anything unrecognized.
func parseAPIError(method, path string, status int, body []byte) *APIError {
	class := classifyStatus(status)
	e := &APIError{StatusCode: status, Method: method, Path: path, Class: class}

	trimmed := strings.TrimSpace(string(body))

	// Empty body (e.g. 401) or HTML page (e.g. Flask 500) → generic, never echoed.
	// Server errors (5xx) always use the generic message regardless of body, so
	// internal details never leak.
	if trimmed == "" || looksLikeHTML(trimmed) || class == ClassServer {
		e.Message = genericMessage(status, class)
		return e
	}

	// Shape: {"detail": ...} — string or array (FastAPI/Pydantic).
	var detailEnv struct {
		Detail json.RawMessage `json:"detail"`
	}
	if err := json.Unmarshal(body, &detailEnv); err == nil && len(detailEnv.Detail) > 0 {
		// detail as array of field errors
		var details []pydanticDetail
		if err := json.Unmarshal(detailEnv.Detail, &details); err == nil && len(details) > 0 {
			msgs := make([]string, 0, len(details))
			for _, d := range details {
				attr := stripLocPrefix(d.Loc)
				e.FieldErrors = append(e.FieldErrors, FieldError{Attribute: attr, Detail: d.Msg})
				if attr != "" {
					msgs = append(msgs, fmt.Sprintf("%s: %s", attr, d.Msg))
				} else {
					msgs = append(msgs, d.Msg)
				}
			}
			e.Message = strings.Join(msgs, "; ")
			return e
		}
		// detail as a plain string
		var detailStr string
		if err := json.Unmarshal(detailEnv.Detail, &detailStr); err == nil && detailStr != "" {
			if looksInternal(detailStr) {
				e.Message = genericMessage(status, class)
			} else {
				e.Message = detailStr
			}
			return e
		}
	}

	// Shape: {"error_title", "error_detail", "request_id"} — never surface request_id.
	var fw struct {
		ErrorTitle  *string         `json:"error_title"`
		ErrorDetail json.RawMessage `json:"error_detail"`
	}
	if err := json.Unmarshal(body, &fw); err == nil && (fw.ErrorTitle != nil || len(fw.ErrorDetail) > 0) {
		msg := ""
		if fw.ErrorTitle != nil {
			msg = strings.TrimSpace(*fw.ErrorTitle)
		}
		// error_detail is sometimes a string, sometimes a number (e.g. 404) — only
		// use it when it is a non-empty string.
		var detailStr string
		if len(fw.ErrorDetail) > 0 && json.Unmarshal(fw.ErrorDetail, &detailStr) == nil && strings.TrimSpace(detailStr) != "" {
			if msg == "" {
				msg = strings.TrimSpace(detailStr)
			} else {
				msg = msg + ": " + strings.TrimSpace(detailStr)
			}
		}
		if msg == "" || looksInternal(msg) {
			e.Message = genericMessage(status, class)
		} else {
			e.Message = msg
		}
		return e
	}

	// Shape: a bare JSON string body, e.g. "Control not found".
	var bare string
	if err := json.Unmarshal(body, &bare); err == nil && bare != "" {
		if looksInternal(bare) {
			e.Message = genericMessage(status, class)
		} else {
			e.Message = bare
		}
		return e
	}

	// Unrecognized body → safe generic message.
	e.Message = genericMessage(status, class)
	return e
}

// AsAPIError extracts an *APIError from an error chain, if present.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// ErrNotFound marks a "resource does not exist" condition for lookups that cannot
// surface an HTTP 404 directly — e.g. a get implemented as list-then-filter, where
// the list call succeeds (200) and the target id/name simply isn't present. Wrap
// such returns with %w so IsNotFound recognizes them and Read can drop from state.
var ErrNotFound = errors.New("resource not found")

// IsNotFound reports whether err is a missing-resource condition — either an APIError
// with HTTP 404, or an error wrapping ErrNotFound (for list-then-filter lookups).
// Resource Read methods use this to drop the resource from state instead of erroring.
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) {
		return true
	}
	apiErr, ok := AsAPIError(err)
	return ok && apiErr.IsNotFound()
}

// IsConflict reports whether err is an APIError for a conflict (HTTP 409).
func IsConflict(err error) bool {
	apiErr, ok := AsAPIError(err)
	return ok && apiErr.IsConflict()
}

// IsServerError reports whether err is an APIError with a 5xx status. Create/update
// recovery paths use this because the platform sometimes returns a 5xx for a resource
// that was actually created (or already exists).
func IsServerError(err error) bool {
	apiErr, ok := AsAPIError(err)
	return ok && apiErr.StatusCode >= 500
}

// StatusCode returns the HTTP status carried by an APIError in err, or 0 if err is
// not an APIError. Use the Is* helpers where a class fits; this is for exact-code checks.
func StatusCode(err error) int {
	apiErr, ok := AsAPIError(err)
	if !ok {
		return 0
	}
	return apiErr.StatusCode
}
