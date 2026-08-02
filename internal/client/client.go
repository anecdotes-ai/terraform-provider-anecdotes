// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"
)

// AnecdotesClient handles API communication with the Anecdotes platform
type AnecdotesClient struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
	token      string
	tokenExp   time.Time
	mu         sync.RWMutex

	// Serializes updates that rewrite a parent object's list.
	parentLocks keyedMutex
}

// keyedMutex provides one mutex per key, created on first use.
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// lock acquires the mutex for key and returns its unlock function.
func (k *keyedMutex) lock(key string) func() {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = make(map[string]*sync.Mutex)
	}
	m, ok := k.locks[key]
	if !ok {
		m = &sync.Mutex{}
		k.locks[key] = m
	}
	k.mu.Unlock()

	m.Lock()
	return m.Unlock
}

// NewAnecdotesClient creates a new Anecdotes API client
func NewAnecdotesClient(apiKey, apiURL string) (*AnecdotesClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	client := &AnecdotesClient{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}

	// Perform initial token exchange to validate credentials
	if err := client.refreshToken(); err != nil {
		return nil, fmt.Errorf("failed to authenticate with Anecdotes API: %w", err)
	}

	return client, nil
}

// refreshToken exchanges the API key for a JWT Bearer token
func (c *AnecdotesClient) refreshToken() error {
	req, err := http.NewRequest("GET", c.apiURL+"/identity/v1/apikey/exchange", nil)
	if err != nil {
		return fmt.Errorf("failed to create token exchange request: %w", err)
	}

	req.Header.Set("x-anecdotes-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exchange API key: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Redact via the shared parser, but return a PLAIN error on purpose:
		// authentication failures must not carry API error classification. A
		// classified 404/5xx from the identity endpoint would otherwise satisfy
		// IsNotFound/IsServerError in callers — dropping healthy resources from
		// state or triggering create recovery for a request that was never sent.
		apiErr := parseAPIError("GET", "/identity/v1/apikey/exchange", resp.StatusCode, body)
		return fmt.Errorf("API key exchange failed (HTTP %d): %s", resp.StatusCode, apiErr.Message)
	}

	// The response is the JWT token as plain text
	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %w", err)
	}

	c.mu.Lock()
	c.token = string(tokenBytes)
	// Token is valid for 60 minutes, refresh 5 minutes before expiry
	c.tokenExp = time.Now().Add(55 * time.Minute)
	c.mu.Unlock()

	return nil
}

// getToken returns a valid JWT token, refreshing if necessary
func (c *AnecdotesClient) getToken() (string, error) {
	c.mu.RLock()
	if time.Now().Before(c.tokenExp) {
		token := c.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	if err := c.refreshToken(); err != nil {
		return "", err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token, nil
}

// isRetryable returns true when a failed request may be safely re-sent.
// 429 is always retryable: the server rejected the request without executing
// it. Server errors (500/502/503) are retried only for idempotent methods —
// re-sending a POST after an ambiguous 5xx could execute the operation twice
// (the create endpoints are known to occasionally return 500 AFTER creating).
func isRetryable(method string, statusCode int) bool {
	if statusCode == 429 {
		return true
	}
	idempotent := method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete || method == http.MethodHead
	return idempotent && (statusCode == 500 || statusCode == 502 || statusCode == 503)
}

// maxRetries is the number of retry attempts for retryable errors.
const maxRetries = 3

// doRequest performs an authenticated HTTP request.
// It treats 2xx and 304 (Not Modified) as success.
func (c *AnecdotesClient) doRequest(method, path string, body interface{}) ([]byte, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	var lastErr error
	refreshed401 := false
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var reqBody io.Reader
		if jsonBody != nil {
			reqBody = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequest(method, c.apiURL+path, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == 304 {
			return respBody, nil
		}

		// A 401 mid-session means the token was revoked or expired before our
		// time-based refresh window. Force a refresh and retry once.
		if resp.StatusCode == http.StatusUnauthorized && !refreshed401 {
			refreshed401 = true
			if rerr := c.refreshToken(); rerr == nil {
				if newToken, terr := c.getToken(); terr == nil {
					token = newToken
				}
			}
			continue
		}

		lastErr = parseAPIError(method, path, resp.StatusCode, respBody)

		if !isRetryable(method, resp.StatusCode) || attempt >= maxRetries {
			return nil, lastErr
		}

		// Exponential backoff: 2s, 4s, 6s
		backoff := time.Duration(attempt+1) * 2 * time.Second
		// Respect Retry-After header if present (for 429)
		if resp.StatusCode == 429 {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := time.ParseDuration(ra + "s"); err == nil {
					backoff = secs
				}
			}
		}
		time.Sleep(backoff)
	}

	return nil, lastErr
}

// doFormRequest performs an authenticated HTTP request with multipart/form-data body.
// The Anecdotes compliance API expects FormData (multipart), NOT application/x-www-form-urlencoded.
// The data map values are strings sent as form fields.
// Includes retry with exponential backoff, subject to isRetryable.
func (c *AnecdotesClient) doFormRequest(method, path string, data map[string]string) ([]byte, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	var lastErr error
	refreshed401 := false
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Rebuild form body on each attempt (reader is consumed)
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for k, v := range data {
			if err := writer.WriteField(k, v); err != nil {
				return nil, fmt.Errorf("failed to write form field %s: %w", k, err)
			}
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close multipart writer: %w", err)
		}

		req, err := http.NewRequest(method, c.apiURL+path, &body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
				continue
			}
			return nil, lastErr
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// A 401 mid-session means the token was revoked or expired before our
		// time-based refresh window. Force a refresh and retry once.
		if resp.StatusCode == http.StatusUnauthorized && !refreshed401 {
			refreshed401 = true
			if rerr := c.refreshToken(); rerr == nil {
				if newToken, terr := c.getToken(); terr == nil {
					token = newToken
				}
			}
			continue
		}

		lastErr = parseAPIError(method, path, resp.StatusCode, respBody)

		if !isRetryable(method, resp.StatusCode) || attempt >= maxRetries {
			return nil, lastErr
		}

		backoff := time.Duration(attempt+1) * 2 * time.Second
		if resp.StatusCode == 429 {
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := time.ParseDuration(ra + "s"); err == nil {
					backoff = secs
				}
			}
		}
		time.Sleep(backoff)
	}

	return nil, lastErr
}

// ListFrameworks retrieves all frameworks
func (c *AnecdotesClient) ListFrameworks() ([]Framework, error) {
	respBody, err := c.doRequest("GET", "/api/v1/framework", nil)
	if err != nil {
		return nil, err
	}

	var frameworks []Framework
	if err := json.Unmarshal(respBody, &frameworks); err != nil {
		return nil, fmt.Errorf("failed to parse frameworks response: %w", err)
	}

	return frameworks, nil
}

// GetFramework retrieves a specific framework by ID
func (c *AnecdotesClient) GetFramework(frameworkID string) (*Framework, error) {
	frameworks, err := c.ListFrameworks()
	if err != nil {
		return nil, err
	}

	for _, fw := range frameworks {
		if fw.FrameworkID == frameworkID {
			return &fw, nil
		}
	}

	return nil, fmt.Errorf("framework not found: %s: %w", frameworkID, ErrNotFound)
}

// CreateFramework creates a new custom framework. When the create call returns
// a server error, the framework is resolved by name; a conflict is returned to
// the caller so an existing framework is never adopted.
func (c *AnecdotesClient) CreateFramework(framework *FrameworkCreateRequest) (*Framework, error) {
	respBody, err := c.doRequest("POST", "/api/v1/framework", framework)
	if err != nil {
		// A conflict is returned as-is: adopting a framework that predates this
		// create would take ownership of an object Terraform did not create.
		if IsServerError(err) {
			if existing, lErr := c.getFrameworkByName(framework.FrameworkName); lErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}

	var frameworkID string
	if err := json.Unmarshal(respBody, &frameworkID); err != nil {
		var result Framework
		if err2 := json.Unmarshal(respBody, &result); err2 != nil {
			return c.getFrameworkByName(framework.FrameworkName)
		}
		return &result, nil
	}

	// Fetch the full framework details using the returned ID
	return c.GetFramework(frameworkID)
}

// getFrameworkByName finds a framework by its name. Used to resolve a framework
// this client created when the create response cannot be used directly.
func (c *AnecdotesClient) getFrameworkByName(name string) (*Framework, error) {
	frameworks, err := c.ListFrameworks()
	if err != nil {
		return nil, fmt.Errorf("framework lookup by name failed: %w", err)
	}
	for _, fw := range frameworks {
		if fw.FrameworkName == name {
			return &fw, nil
		}
	}
	return nil, fmt.Errorf("framework %q not found in list: %w", name, ErrNotFound)
}

// SetFrameworkAuditorControlStatus sets the auditor-visible control statuses via
// the dedicated endpoint (the framework create/update body rejects this field).
func (c *AnecdotesClient) SetFrameworkAuditorControlStatus(frameworkID string, status *FrameworkAuditorControlStatus) error {
	_, err := c.doRequest("PUT", "/api/v1/framework/"+frameworkID+"/auditor_control_status", status)
	return err
}

// SetFrameworkAuditorEvidenceStatus sets the auditor-visible evidence statuses.
func (c *AnecdotesClient) SetFrameworkAuditorEvidenceStatus(frameworkID string, status *FrameworkAuditorEvidenceStatus) error {
	_, err := c.doRequest("PUT", "/api/v1/framework/"+frameworkID+"/auditor_evidence_status", status)
	return err
}

// UpdateFramework updates an existing framework
func (c *AnecdotesClient) UpdateFramework(frameworkID string, framework *FrameworkUpdateRequest) (*Framework, error) {
	respBody, err := c.doRequest("PATCH", "/api/v1/framework/"+frameworkID, framework)
	if err != nil {
		return nil, err
	}

	// If response is empty, fetch the updated framework
	if len(respBody) == 0 {
		return c.GetFramework(frameworkID)
	}

	var result Framework
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Fallback: try fetching if parsing fails
		return c.GetFramework(frameworkID)
	}

	return &result, nil
}

// DeleteFramework deletes a framework
func (c *AnecdotesClient) DeleteFramework(frameworkID string) error {
	_, err := c.doRequest("DELETE", "/api/v1/framework/"+frameworkID, nil)
	return err
}

// ListControlCategories retrieves all control categories
func (c *AnecdotesClient) ListControlCategories() ([]ControlCategory, error) {
	respBody, err := c.doRequest("GET", "/api/v1/framework/category", nil)
	if err != nil {
		return nil, err
	}

	var categories []ControlCategory
	if err := json.Unmarshal(respBody, &categories); err != nil {
		return nil, fmt.Errorf("failed to parse categories response: %w", err)
	}

	return categories, nil
}

// CreateControlCategory creates a new control category. When the create call
// returns a server error, the category is resolved by name within the same
// framework; a conflict is returned to the caller so an existing category is
// never adopted (use `terraform import` for that).
func (c *AnecdotesClient) CreateControlCategory(category *ControlCategoryCreateRequest) (*ControlCategory, error) {
	respBody, err := c.doRequest("POST", "/api/v1/framework/category", category)
	if err != nil {
		if IsServerError(err) {
			existing, lErr := c.getControlCategoryByName(category.CategoryName, category.FrameworkID)
			if lErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}

	// Response might be the category ID as string or the full object
	var categoryID string
	if err := json.Unmarshal(respBody, &categoryID); err == nil && categoryID != "" {
		// Return with the ID we got
		return &ControlCategory{
			CategoryID:   categoryID,
			CategoryName: category.CategoryName,
			FrameworkID:  category.FrameworkID,
		}, nil
	}

	// Try parsing as full category object
	var result ControlCategory
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create category response: %w", err)
	}

	return &result, nil
}

// getControlCategoryByName finds a control category by name within a specific
// framework (used to recover our own creation after an ambiguous 5xx).
// The match is scoped to the requested framework on purpose: matching a
// same-named category from another framework would adopt an object this
// create did not produce.
func (c *AnecdotesClient) getControlCategoryByName(name, frameworkID string) (*ControlCategory, error) {
	categories, err := c.ListControlCategories()
	if err != nil {
		return nil, fmt.Errorf("category lookup by name failed: %w", err)
	}
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for _, cat := range categories {
		if strings.ToLower(strings.TrimSpace(cat.CategoryName)) == nameLower && cat.FrameworkID == frameworkID {
			return &cat, nil
		}
	}
	return nil, fmt.Errorf("category %q not found in framework %s (%d total): %w", name, frameworkID, len(categories), ErrNotFound)
}

// UpdateControlCategory updates an existing control category
func (c *AnecdotesClient) UpdateControlCategory(categoryID string, category *ControlCategoryUpdateRequest) error {
	_, err := c.doRequest("PATCH", "/api/v1/framework/category/"+categoryID, category)
	return err
}

// DeleteControlCategory deletes a control category
func (c *AnecdotesClient) DeleteControlCategory(categoryID string) error {
	_, err := c.doRequest("DELETE", "/api/v1/framework/category/"+categoryID, nil)
	return err
}

// GetControlCategory retrieves a specific category by searching through all categories
func (c *AnecdotesClient) GetControlCategory(categoryID string) (*ControlCategory, error) {
	categories, err := c.ListControlCategories()
	if err != nil {
		return nil, err
	}

	for _, cat := range categories {
		if cat.CategoryID == categoryID {
			return &cat, nil
		}
	}

	return nil, fmt.Errorf("category not found: %s: %w", categoryID, ErrNotFound)
}

// GetControl retrieves a specific control by ID
func (c *AnecdotesClient) GetControl(frameworkID, controlID string) (*Control, error) {
	// Use POST /control/read to get control by ID
	body := map[string][]string{
		"controls_ids": {controlID},
	}
	respBody, err := c.doRequest("POST", "/controls/control/read", body)
	if err != nil {
		return nil, err
	}

	var results []Control
	if err := json.Unmarshal(respBody, &results); err != nil {
		return nil, fmt.Errorf("failed to parse control response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("control not found: %s: %w", controlID, ErrNotFound)
	}

	return &results[0], nil
}

// ListControls retrieves all controls for a framework.
// Uses the controls service path /controls/framework_controls (per OpenAPI).
func (c *AnecdotesClient) ListControls(frameworkID string) ([]Control, error) {
	respBody, err := c.doRequest("GET", "/controls/framework_controls?frameworks_ids="+frameworkID, nil)
	if err != nil {
		return nil, err
	}

	// Response is a map: framework_id -> []Control
	var byFramework map[string][]Control
	if err := json.Unmarshal(respBody, &byFramework); err != nil {
		return nil, fmt.Errorf("failed to parse controls response: %w", err)
	}

	controls := byFramework[frameworkID]
	return controls, nil
}

// AddControl adds a single control to a framework.
// Retries on "nonexistent category" errors (API propagation delay).
func (c *AnecdotesClient) AddControl(frameworkID string, control *ControlCreateRequest) (*Control, error) {
	respBody, err := c.doRequest("POST", "/controls/control?control_framework="+frameworkID, control)
	if err != nil {
		errMsg := err.Error()
		// Retry on "nonexistent category" — the category was just created but may not be propagated yet
		if strings.Contains(errMsg, "nonexistent category") || strings.Contains(errMsg, "nonexistent") {
			for retry := 0; retry < 3; retry++ {
				time.Sleep(time.Duration(3+retry*3) * time.Second)
				respBody, err = c.doRequest("POST", "/controls/control?control_framework="+frameworkID, control)
				if err == nil {
					break
				}
				if !strings.Contains(err.Error(), "nonexistent") {
					return nil, err
				}
			}
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// The API may return just the control ID as a string
	var controlID string
	if err := json.Unmarshal(respBody, &controlID); err == nil && controlID != "" {
		// Fetch the full control details
		return c.GetControl(frameworkID, controlID)
	}

	// Try parsing as full Control object
	var result Control
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create control response: %w", err)
	}

	return &result, nil
}

// UpdateControl updates an existing control
func (c *AnecdotesClient) UpdateControl(frameworkID, controlID string, control *ControlUpdateRequest) (*Control, error) {
	respBody, err := c.doRequest("PUT", "/controls/control/"+controlID, control)
	if err != nil {
		return nil, err
	}

	// Response may be empty on success, fetch updated control
	if len(respBody) == 0 || string(respBody) == "{}" {
		return c.GetControl(frameworkID, controlID)
	}

	var result Control
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If parsing fails, try to fetch the updated control
		return c.GetControl(frameworkID, controlID)
	}

	return &result, nil
}

// DeleteControl deletes a control from a framework
func (c *AnecdotesClient) DeleteControl(frameworkID, controlID string) error {
	_, err := c.doRequest("DELETE", "/controls/control/"+controlID, nil)
	return err
}

// ImportControls imports controls into a framework via bulk import
func (c *AnecdotesClient) ImportControls(frameworkID string, controls []ControlImport) error {
	body := map[string]interface{}{
		"controls": controls,
	}
	_, err := c.doRequest("POST", "/api/v1/framework/"+frameworkID+"/controls/import", body)
	return err
}

// SetControlOwners replaces a control's owners. An empty slice clears them.
func (c *AnecdotesClient) SetControlOwners(controlID string, owners []string) error {
	if owners == nil {
		owners = []string{}
	}

	body := []map[string]interface{}{
		{
			"control_id":    controlID,
			"control_owner": owners,
		},
	}

	_, err := c.doRequest("PATCH", "/controls/controls", body)
	return err
}

// SetControlMaturityLevel sets the maturity level for one or more controls
// An empty level clears the control's maturity level.
func (c *AnecdotesClient) SetControlMaturityLevel(controlID, maturityLevel string) error {
	var level interface{}
	if maturityLevel != "" {
		level = maturityLevel
	}

	body := map[string]interface{}{
		"controls": []map[string]interface{}{
			{
				"control_id":     controlID,
				"maturity_level": level,
			},
		},
	}

	_, err := c.doRequest("PATCH", "/controls/controls/maturity_level", body)
	return err
}

// GetRequirement retrieves a specific requirement by ID
func (c *AnecdotesClient) GetRequirement(requirementID string) (*Requirement, error) {
	respBody, err := c.doRequest("GET", "/api/v1/requirement/"+requirementID, nil)
	if err != nil {
		return nil, err
	}

	// API may return an array even for single ID
	var results []Requirement
	if err := json.Unmarshal(respBody, &results); err == nil {
		// An empty array means the requirement does not exist (mirrors GetControl).
		if len(results) == 0 {
			return nil, fmt.Errorf("requirement not found: %s: %w", requirementID, ErrNotFound)
		}
		c.resolveRequirementStatusNames([]*Requirement{&results[0]})
		return &results[0], nil
	}

	// Try parsing as single object
	var result Requirement
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse requirement response: %w", err)
	}

	c.resolveRequirementStatusNames([]*Requirement{&result})
	return &result, nil
}

// ListRequirements retrieves all requirements from the Requirements Hub.
// GET /api/v1/requirement (no id) returns all requirement instances per OpenAPI.
func (c *AnecdotesClient) ListRequirements() ([]Requirement, error) {
	respBody, err := c.doRequest("GET", "/api/v1/requirement", nil)
	if err != nil {
		return nil, err
	}

	var requirements []Requirement
	if err := json.Unmarshal(respBody, &requirements); err != nil {
		return nil, fmt.Errorf("failed to parse requirements response: %w", err)
	}

	ptrs := make([]*Requirement, len(requirements))
	for i := range requirements {
		ptrs[i] = &requirements[i]
	}
	c.resolveRequirementStatusNames(ptrs)

	return requirements, nil
}

// CreateRequirement creates a new requirement in the Requirements Hub. The
// second return value reports whether the requirement was resolved by name
// after a server error rather than read back from the create call.
func (c *AnecdotesClient) CreateRequirement(requirement *RequirementCreateRequest) (*Requirement, bool, error) {
	respBody, err := c.doRequest("POST", "/api/v1/requirement", requirement)
	if err != nil {
		// A conflict is returned as-is: adopting a requirement that predates this
		// create would take ownership of an object Terraform did not create.
		if IsServerError(err) {
			existing, lErr := c.getRequirementByName(requirement.RequirementDescription)
			if lErr == nil && existing != nil {
				return existing, true, nil
			}
		}
		return nil, false, err
	}

	var requirementID string
	if err := json.Unmarshal(respBody, &requirementID); err == nil && requirementID != "" {
		created, err := c.GetRequirement(requirementID)
		return created, false, err
	}

	var result Requirement
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, false, fmt.Errorf("failed to parse create requirement response: %w", err)
	}

	return &result, false, nil
}

// getRequirementByName finds a requirement by its description/name. Used to
// resolve a requirement this client created when the create call returned an error.
func (c *AnecdotesClient) getRequirementByName(name string) (*Requirement, error) {
	requirements, err := c.ListRequirements()
	if err != nil {
		return nil, fmt.Errorf("requirement create failed and list fallback also failed: %w", err)
	}
	nameLower := strings.ToLower(strings.TrimSpace(name))
	for _, r := range requirements {
		if strings.ToLower(strings.TrimSpace(r.RequirementDescription)) == nameLower {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("requirement create failed and requirement %q not found in list (%d total)", name, len(requirements))
}

// UpdateRequirement updates an existing requirement.
// The API requires the request body wrapped in {"requirement": {...}}.
func (c *AnecdotesClient) UpdateRequirement(requirementID string, requirement *RequirementUpdateRequest) (*Requirement, error) {
	wrapped := map[string]interface{}{"requirement": requirement}
	_, err := c.doRequest("PATCH", "/api/v1/requirement/"+requirementID, wrapped)
	if err != nil {
		return nil, err
	}

	// Fetch updated requirement
	return c.GetRequirement(requirementID)
}

// DeleteRequirement deletes a requirement via POST /api/v1/requirement/delete.
// DELETE /api/v1/requirement/{id} returns 405; the FE uses this POST endpoint instead.
func (c *AnecdotesClient) DeleteRequirement(requirementID string) error {
	body := []string{requirementID}
	_, err := c.doRequest("POST", "/api/v1/requirement/delete", body)
	return err
}

// ListRequirementStatuses fetches the available requirement status options.
// API: GET /api/v1/requirement/status
// Returns a map of status name → status ID.
func (c *AnecdotesClient) ListRequirementStatuses() (map[string]string, error) {
	respBody, err := c.doRequest("GET", "/api/v1/requirement/status", nil)
	if err != nil {
		return nil, err
	}

	var statuses []struct {
		Name     string `json:"name"`
		StatusID string `json:"status_id"`
	}
	if err := json.Unmarshal(respBody, &statuses); err != nil {
		return nil, fmt.Errorf("failed to parse requirement statuses: %w", err)
	}

	result := make(map[string]string, len(statuses))
	for _, s := range statuses {
		result[s.Name] = s.StatusID
	}
	return result, nil
}

// requirementStatusNameByID returns a status-ID → status-name map (the inverse of
// ListRequirementStatuses, which returns name → ID). Returns nil on lookup failure.
func (c *AnecdotesClient) requirementStatusNameByID() map[string]string {
	statusMap, err := c.ListRequirementStatuses()
	if err != nil {
		return nil
	}
	inv := make(map[string]string, len(statusMap))
	for name, id := range statusMap {
		inv[id] = name
	}
	return inv
}

// resolveRequirementStatusNames fills the human-readable status fields from the
// requirement_status_id the API returns. The requirement read endpoints return only
// the status ID, so RequirementStatusName / RequirementStatus would otherwise be
// empty. Best-effort: requirements are left unchanged if the status list can't be
// fetched, and only one statuses lookup is performed for the whole batch.
func (c *AnecdotesClient) resolveRequirementStatusNames(reqs []*Requirement) {
	needsResolve := false
	for _, r := range reqs {
		if r.RequirementStatusName == "" && r.RequirementStatusID != "" {
			needsResolve = true
			break
		}
	}
	if !needsResolve {
		return
	}

	nameByID := c.requirementStatusNameByID()
	if nameByID == nil {
		return
	}

	for _, r := range reqs {
		if r.RequirementStatusName != "" || r.RequirementStatusID == "" {
			continue
		}
		if name, ok := nameByID[r.RequirementStatusID]; ok {
			r.RequirementStatusName = name
			// RequirementStatus backs the data-source "status" attribute and the
			// requirements list-filter, both of which compare against the name.
			if r.RequirementStatus == "" {
				r.RequirementStatus = name
			}
		}
	}
}

// LinkRequirementToControl links a requirement to a control using read-modify-write
// via PATCH /controls/controls (the dedicated link endpoint returns 404).
func (c *AnecdotesClient) LinkRequirementToControl(controlID, requirementID string) (*ControlRequirementLink, error) {
	defer c.parentLocks.lock("control:" + controlID)()

	// Read current control to get existing requirement IDs
	control, err := c.GetControl("", controlID)
	if err != nil {
		return nil, fmt.Errorf("failed to read control %s: %w", controlID, err)
	}

	// Check if already linked
	for _, rid := range control.RequirementIDs {
		if rid == requirementID {
			return &ControlRequirementLink{
				ControlID:     controlID,
				RequirementID: requirementID,
				FrameworkID:   control.FrameworkID,
			}, nil
		}
	}

	// Append the new requirement and update
	updatedIDs := append(control.RequirementIDs, requirementID)
	if err := c.UpdateControlRequirements(controlID, updatedIDs); err != nil {
		return nil, fmt.Errorf("failed to link requirement %s to control %s: %w", requirementID, controlID, err)
	}

	return &ControlRequirementLink{
		ControlID:     controlID,
		RequirementID: requirementID,
		FrameworkID:   control.FrameworkID,
	}, nil
}

// UnlinkRequirementFromControl removes a requirement link from a control using
// read-modify-write via PATCH /controls/controls.
func (c *AnecdotesClient) UnlinkRequirementFromControl(controlID, requirementID string) error {
	defer c.parentLocks.lock("control:" + controlID)()

	// Read current control to get existing requirement IDs
	control, err := c.GetControl("", controlID)
	if err != nil {
		return fmt.Errorf("failed to read control %s: %w", controlID, err)
	}

	updatedIDs := make([]string, 0, len(control.RequirementIDs))
	for _, rid := range control.RequirementIDs {
		if rid != requirementID {
			updatedIDs = append(updatedIDs, rid)
		}
	}

	// If nothing changed, the requirement wasn't linked
	if len(updatedIDs) == len(control.RequirementIDs) {
		return nil // Already unlinked, no-op
	}

	return c.UpdateControlRequirements(controlID, updatedIDs)
}

// GetControlRequirementLink verifies a link exists between a control and requirement.
// Uses POST /controls/control/read to read control_requirement_ids from the control.
func (c *AnecdotesClient) GetControlRequirementLink(controlID, requirementID string) (*ControlRequirementLink, error) {
	control, err := c.GetControl("", controlID)
	if err != nil {
		return nil, fmt.Errorf("failed to read control %s: %w", controlID, err)
	}

	// Check if the requirement is in the control's requirement IDs
	for _, rid := range control.RequirementIDs {
		if rid == requirementID {
			return &ControlRequirementLink{
				ControlID:     controlID,
				RequirementID: requirementID,
				FrameworkID:   control.FrameworkID,
			}, nil
		}
	}

	return nil, fmt.Errorf("requirement %s is not linked to control %s: %w", requirementID, controlID, ErrNotFound)
}

// UpdateControlRequirements updates the requirements linked to a control
func (c *AnecdotesClient) UpdateControlRequirements(controlID string, requirementIDs []string) error {
	payload := []map[string]interface{}{
		{
			"control_id":                   controlID,
			"control_related_requirements": requirementIDs,
			"requirements_recommendations": []string{},
		},
	}

	_, err := c.doRequest("PATCH", "/controls/controls", payload)
	if err != nil {
		return fmt.Errorf("failed to update control requirements: %w", err)
	}

	return nil
}

// CreateFolder creates a new folder for organizing frameworks
func (c *AnecdotesClient) CreateFolder(folder *FolderCreateRequest) (*Folder, error) {
	respBody, err := c.doRequest("POST", "/frameworks/v1/folders", folder)
	if err != nil {
		return nil, err
	}

	var result Folder
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If we can't parse, return with the ID we sent
		return &Folder{
			ID:             folder.ID,
			Name:           folder.Name,
			FrameworksList: folder.FrameworksList,
		}, nil
	}

	return &result, nil
}

// GetFolder retrieves a specific folder by ID
func (c *AnecdotesClient) GetFolder(folderID string) (*Folder, error) {
	respBody, err := c.doRequest("GET", "/frameworks/v1/folders/"+folderID, nil)
	if err != nil {
		return nil, err
	}

	var result Folder
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse folder response: %w", err)
	}

	return &result, nil
}

// ListFolders retrieves all folders
func (c *AnecdotesClient) ListFolders() ([]Folder, error) {
	respBody, err := c.doRequest("GET", "/frameworks/v1/folders", nil)
	if err != nil {
		return nil, err
	}

	var folders []Folder
	if err := json.Unmarshal(respBody, &folders); err != nil {
		return nil, fmt.Errorf("failed to parse folders response: %w", err)
	}

	return folders, nil
}

// UpdateFolder updates an existing folder
func (c *AnecdotesClient) UpdateFolder(folderID string, folder *FolderUpdateRequest) (*Folder, error) {
	respBody, err := c.doRequest("PATCH", "/frameworks/v1/folders/"+folderID, folder)
	if err != nil {
		return nil, err
	}

	// If response is empty, fetch the updated folder
	if len(respBody) == 0 {
		return c.GetFolder(folderID)
	}

	var result Folder
	if err := json.Unmarshal(respBody, &result); err != nil {
		return c.GetFolder(folderID)
	}

	return &result, nil
}

// DeleteFolder deletes a folder
func (c *AnecdotesClient) DeleteFolder(folderID string) error {
	_, err := c.doRequest("DELETE", "/frameworks/v1/folders/"+folderID, nil)
	return err
}

// FindFrameworkFolder returns the ID of the folder containing the framework,
// or "" when the framework is not in any folder.
func (c *AnecdotesClient) FindFrameworkFolder(frameworkID string) (string, error) {
	folders, err := c.ListFolders()
	if err != nil {
		return "", err
	}
	for _, folder := range folders {
		for _, id := range folder.FrameworksList {
			if id == frameworkID {
				return folder.ID, nil
			}
		}
	}
	return "", nil
}

// MoveFrameworkFolder moves a framework between folders. Both folder IDs must
// reference existing folders.
func (c *AnecdotesClient) MoveFrameworkFolder(frameworkID, fromFolderID, toFolderID string) error {
	body := map[string]string{
		"remove_from_folder_id": fromFolderID,
		"add_to_folder_id":      toFolderID,
	}
	_, err := c.doRequest("PUT", "/frameworks/v1/folders/framework/"+frameworkID, body)
	return err
}

// ListEvidences returns all evidences in the account.
// GET /evidence/v1/evidence returns a flat JSON array of Evidence objects.
func (c *AnecdotesClient) ListEvidences() ([]Evidence, error) {
	respBody, err := c.doRequest("GET", "/evidence/v1/evidence", nil)
	if err != nil {
		return nil, err
	}

	var evidences []Evidence
	if err := json.Unmarshal(respBody, &evidences); err != nil {
		return nil, fmt.Errorf("failed to parse evidences list response: %w", err)
	}

	return evidences, nil
}

// GetAttachmentUploadURL requests a presigned upload URL for evidence file upload.
// POST /v1/attachments/upload-url (note: NOT under /evidence/v1, uses a separate base path).
func (c *AnecdotesClient) GetAttachmentUploadURL(filename string, fileSize int64, contentType string) (*AttachmentUploadResponse, error) {
	reqBody := AttachmentUploadRequest{
		Context: AttachmentContext{Type: "evidence"},
		File: AttachmentFile{
			Filename:    filename,
			FileSize:    fileSize,
			ContentType: contentType,
		},
	}

	// This endpoint is at /v1/attachments/upload-url (NOT /evidence/v1)
	respBody, err := c.doRequest("POST", "/v1/attachments/upload-url", reqBody)
	if err != nil {
		return nil, err
	}

	var result AttachmentUploadResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse attachment upload response: %w", err)
	}

	return &result, nil
}

// UploadFileToURL uploads raw file bytes to a presigned URL (e.g., GCS).
func (c *AnecdotesClient) UploadFileToURL(uploadURL string, fileData []byte, contentType string) error {
	req, err := http.NewRequest("PUT", uploadURL, bytes.NewReader(fileData))
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("file upload failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Do not echo the storage backend's response body (raw XML/HTML).
		return fmt.Errorf("file upload failed with status %d", resp.StatusCode)
	}

	return nil
}

// CreateEvidenceFile creates a new file-type (MANUAL) evidence item via the 3-step upload flow.
// 1. Get presigned upload URL
// 2. Upload file to the URL
// 3. Create evidence record with attachment_id
func (c *AnecdotesClient) CreateEvidenceFile(evidenceName string, fileData []byte, filename string, contentType string) (string, error) {
	// Step 1: Get presigned upload URL
	uploadResp, err := c.GetAttachmentUploadURL(filename, int64(len(fileData)), contentType)
	if err != nil {
		return "", fmt.Errorf("failed to get upload URL: %w", err)
	}

	// Step 2: Upload file to presigned URL
	if err := c.UploadFileToURL(uploadResp.UploadURL, fileData, contentType); err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Step 3: Create evidence record with attachment_id
	formData := map[string]string{
		"attachment_id":      uploadResp.AttachmentID,
		"uploaded_to_bucket": "true",
		"evidence_name":      evidenceName,
	}

	respBody, err := c.doFormRequest("POST", "/evidence/v1/evidence/manual", formData)
	if err != nil {
		return "", fmt.Errorf("failed to create evidence record: %w", err)
	}

	var result EvidenceCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse create evidence response: %w", err)
	}

	if len(result.EvidenceID) == 0 {
		return "", fmt.Errorf("create evidence returned empty ID array")
	}

	return result.EvidenceID[0], nil
}

// LinkEvidenceToRequirement adds an evidence ID to a requirement's evidence list.
// Uses read-modify-write: reads current list, appends if missing, PATCHes back.
// The API uses `requirement_related_evidences` for write and `requirement_evidence_ids` for read.
func (c *AnecdotesClient) LinkEvidenceToRequirement(requirementID, evidenceID string) error {
	defer c.parentLocks.lock("requirement:" + requirementID)()

	req, err := c.GetRequirement(requirementID)
	if err != nil {
		return fmt.Errorf("failed to read requirement %s: %w", requirementID, err)
	}

	// Check if already linked (read field is RequirementEvidenceIDs)
	for _, eid := range req.RequirementEvidenceIDs {
		if eid == evidenceID {
			return nil // Already linked, nothing to do
		}
	}

	// Append and update (write field is RequirementRelatedEvidences)
	updatedEvidences := append(req.RequirementEvidenceIDs, evidenceID)
	updateReq := &RequirementUpdateRequest{
		RequirementRelatedEvidences: &updatedEvidences,
	}

	_, err = c.UpdateRequirement(requirementID, updateReq)
	return err
}

// UnlinkEvidenceFromRequirement removes an evidence ID from a requirement's evidence list.
func (c *AnecdotesClient) UnlinkEvidenceFromRequirement(requirementID, evidenceID string) error {
	defer c.parentLocks.lock("requirement:" + requirementID)()

	req, err := c.GetRequirement(requirementID)
	if err != nil {
		return fmt.Errorf("failed to read requirement %s: %w", requirementID, err)
	}

	// Filter out the evidence ID. The slice starts non-nil so removing the
	// LAST evidence still serializes as an empty list instead of being
	// dropped by omitempty (which would silently leave the link in place).
	updatedEvidences := make([]string, 0, len(req.RequirementEvidenceIDs))
	for _, eid := range req.RequirementEvidenceIDs {
		if eid != evidenceID {
			updatedEvidences = append(updatedEvidences, eid)
		}
	}

	// If nothing changed, the evidence wasn't linked
	if len(updatedEvidences) == len(req.RequirementEvidenceIDs) {
		return nil // Wasn't linked, nothing to do
	}

	updateReq := &RequirementUpdateRequest{
		RequirementRelatedEvidences: &updatedEvidences,
	}

	_, err = c.UpdateRequirement(requirementID, updateReq)
	return err
}

// GetRequirementEvidenceLink checks if an evidence ID is linked to a requirement.
// Returns nil if linked, error if not found.
func (c *AnecdotesClient) GetRequirementEvidenceLink(requirementID, evidenceID string) error {
	req, err := c.GetRequirement(requirementID)
	if err != nil {
		return fmt.Errorf("failed to read requirement %s: %w", requirementID, err)
	}

	for _, eid := range req.RequirementEvidenceIDs {
		if eid == evidenceID {
			return nil // Found
		}
	}

	return fmt.Errorf("evidence %s is not linked to requirement %s: %w", evidenceID, requirementID, ErrNotFound)
}

// GetEvidenceFullData retrieves the full table data for an evidence instance.
// GET /evidence/v1/evidence/{instance_id}/full_data
// Returns column-oriented data: {"ColumnName": ["val1", "val2", ...], ...}
func (c *AnecdotesClient) GetEvidenceFullData(instanceID string) (map[string]interface{}, error) {
	respBody, err := c.doRequest("GET", "/evidence/v1/evidence/"+instanceID+"/full_data", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse full data response: %w", err)
	}

	// Check for error response
	if errTitle, ok := result["error_title"]; ok {
		return nil, fmt.Errorf("full data error: %v", errTitle)
	}

	return result, nil
}

// GetControlMaturityLevel reads the maturity level from a control.
// Note: SetControlMaturityLevel already exists above (line ~449).
func (c *AnecdotesClient) GetControlMaturityLevel(controlID string) (string, error) {
	respBody, err := c.doRequest("POST", "/controls/control/read", map[string][]string{
		"controls_ids": {controlID},
	})
	if err != nil {
		return "", err
	}

	var rawResults []map[string]interface{}
	if err := json.Unmarshal(respBody, &rawResults); err != nil {
		return "", fmt.Errorf("failed to parse control response: %w", err)
	}

	if len(rawResults) == 0 {
		return "", fmt.Errorf("control not found: %s: %w", controlID, ErrNotFound)
	}

	if ml, ok := rawResults[0]["maturity_level"]; ok && ml != nil {
		return normalizeMaturityLevel(fmt.Sprintf("%v", ml)), nil
	}

	return "", nil
}

// normalizeMaturityLevel returns raw as one of the valid maturity levels, or
// "" when it is not one of them.
func normalizeMaturityLevel(raw string) string {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	for _, l := range ValidMaturityLevels() {
		if upper == l {
			return l
		}
	}
	return ""
}
