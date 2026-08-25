// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

	// Serializes token refreshes so concurrent callers exchange the key once.
	refreshMu sync.Mutex

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

// errRedirectRefused marks a refused redirect. It can never succeed on retry.
var errRedirectRefused = errors.New("redirect refused")

// NewAnecdotesClient creates a new Anecdotes API client
func NewAnecdotesClient(ctx context.Context, apiKey, apiURL string) (*AnecdotesClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	client := &AnecdotesClient{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			// The API key travels in a custom header, which Go forwards across
			// hosts even though it strips Authorization. Refusing redirects keeps
			// the credential from following one.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return fmt.Errorf("refusing to follow a redirect to %s: the Anecdotes API is not expected to redirect: %w",
					req.URL.Redacted(), errRedirectRefused)
			},
		},
	}

	// Perform initial token exchange to validate credentials
	if err := client.refreshToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to authenticate with Anecdotes API: %w", err)
	}

	return client, nil
}

// refreshToken exchanges the API key for a JWT Bearer token. Transient
// failures are retried; a rejected key is not.
func (c *AnecdotesClient) refreshToken(ctx context.Context) error {
	var lastStatus int
	var lastBody []byte

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", c.apiURL+"/identity/v1/apikey/exchange", nil)
		if err != nil {
			return fmt.Errorf("failed to create token exchange request: %w", err)
		}

		req.Header.Set("x-anecdotes-api-key", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// A refused redirect can never succeed on retry.
			if errors.Is(err, errRedirectRefused) || attempt == maxRetries {
				return fmt.Errorf("failed to exchange API key: %w", err)
			}
			if serr := sleepFor(ctx, time.Duration(attempt+1)*retryBackoffBase); serr != nil {
				return serr
			}
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		status := resp.StatusCode
		retryAfter := resp.Header.Get("Retry-After")
		_ = resp.Body.Close()

		if status == http.StatusOK {
			if readErr != nil {
				return fmt.Errorf("failed to read token response: %w", readErr)
			}
			c.mu.Lock()
			// The response is the JWT token as plain text.
			c.token = string(body)
			// Token is valid for 60 minutes, refresh 5 minutes before expiry
			c.tokenExp = time.Now().Add(55 * time.Minute)
			c.mu.Unlock()
			return nil
		}

		lastStatus, lastBody = status, body

		// A rejected key stays rejected, so only transient statuses are retried.
		transient := status == http.StatusTooManyRequests || status >= 500
		if !transient || attempt == maxRetries {
			break
		}

		backoff := time.Duration(attempt+1) * retryBackoffBase
		if wait, ok := parseRetryAfter(retryAfter, time.Now()); ok {
			backoff = wait
		}
		if serr := sleepFor(ctx, backoff); serr != nil {
			return serr
		}
	}

	// Redact via the shared parser, but return a PLAIN error on purpose:
	// authentication failures must not carry API error classification. A
	// classified 404/5xx from the identity endpoint would otherwise satisfy
	// IsNotFound/IsServerError in callers — dropping healthy resources from
	// state or triggering create recovery for a request that was never sent.
	apiErr := parseAPIError("GET", "/identity/v1/apikey/exchange", lastStatus, lastBody)
	return fmt.Errorf("API key exchange failed (HTTP %d): %s", lastStatus, apiErr.Message)
}

// getToken returns a valid JWT token, refreshing if necessary
func (c *AnecdotesClient) getToken(ctx context.Context) (string, error) {
	if token, ok := c.cachedToken(ctx); ok {
		return token, nil
	}

	// One goroutine refreshes; the rest find the result cached.
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	if token, ok := c.cachedToken(ctx); ok {
		return token, nil
	}

	if err := c.refreshToken(ctx); err != nil {
		return "", err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token, nil
}

// cachedToken returns the current token when it is still valid.
func (c *AnecdotesClient) cachedToken(ctx context.Context) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Now().Before(c.tokenExp) {
		return c.token, true
	}
	return "", false
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

// retryBackoffBase scales the wait between retries: 2s, 4s, 6s.
// A variable so tests can shorten it.
var retryBackoffBase = 2 * time.Second

// maxRetryAfter bounds how long a Retry-After header may pause the provider.
// A variable so tests can shorten it.
var maxRetryAfter = 60 * time.Second

// parseRetryAfter reads a Retry-After header in either form RFC 9110 allows.
// An unparseable value reports false. The result is capped at maxRetryAfter.
func parseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	var wait time.Duration
	if secs, err := strconv.Atoi(value); err == nil {
		// delay-seconds
		wait = time.Duration(secs) * time.Second
	} else {
		// HTTP-date
		at, err := http.ParseTime(value)
		if err != nil {
			return 0, false
		}
		wait = at.Sub(now)
	}

	// Zero or a past date means retry now.
	if wait <= 0 {
		return 0, false
	}
	if wait > maxRetryAfter {
		wait = maxRetryAfter
	}
	return wait, true
}

// sleepFor waits for d, or stops early when ctx is cancelled.
func sleepFor(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// doRequest performs an authenticated HTTP request.
// It treats 2xx and 304 (Not Modified) as success.
func (c *AnecdotesClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	token, err := c.getToken(ctx)
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

		req, err := http.NewRequestWithContext(ctx, method, c.apiURL+path, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if errors.Is(err, errRedirectRefused) {
				return nil, lastErr
			}
			if attempt < maxRetries {
				if serr := sleepFor(ctx, time.Duration(attempt+1)*retryBackoffBase); serr != nil {
					return nil, serr
				}
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
			if rerr := c.refreshToken(ctx); rerr == nil {
				if newToken, terr := c.getToken(ctx); terr == nil {
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
		backoff := time.Duration(attempt+1) * retryBackoffBase
		// Respect Retry-After header if present (for 429)
		if resp.StatusCode == 429 {
			if wait, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); ok {
				backoff = wait
			}
		}
		if serr := sleepFor(ctx, backoff); serr != nil {
			return nil, serr
		}
	}

	return nil, lastErr
}

// ListFrameworks retrieves all frameworks
func (c *AnecdotesClient) ListFrameworks(ctx context.Context) ([]Framework, error) {
	respBody, err := c.doRequest(ctx, "GET", "/api/v1/framework", nil)
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
func (c *AnecdotesClient) GetFramework(ctx context.Context, frameworkID string) (*Framework, error) {
	frameworks, err := c.ListFrameworks(ctx)
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
func (c *AnecdotesClient) CreateFramework(ctx context.Context, framework *FrameworkCreateRequest) (*Framework, error) {
	respBody, err := c.doRequest(ctx, "POST", "/api/v1/framework", framework)
	if err != nil {
		// A conflict is returned as-is: adopting a framework that predates this
		// create would take ownership of an object Terraform did not create.
		if IsServerError(err) {
			if existing, lErr := c.getFrameworkByName(ctx, framework.FrameworkName); lErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}

	var frameworkID string
	if err := json.Unmarshal(respBody, &frameworkID); err != nil {
		var result Framework
		if err2 := json.Unmarshal(respBody, &result); err2 != nil {
			return c.getFrameworkByName(ctx, framework.FrameworkName)
		}
		return &result, nil
	}

	// Fetch the full framework details using the returned ID
	return c.GetFramework(ctx, frameworkID)
}

// getFrameworkByName finds a framework by its name. Used to resolve a framework
// this client created when the create response cannot be used directly.
func (c *AnecdotesClient) getFrameworkByName(ctx context.Context, name string) (*Framework, error) {
	frameworks, err := c.ListFrameworks(ctx)
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
func (c *AnecdotesClient) SetFrameworkAuditorControlStatus(ctx context.Context, frameworkID string, status *FrameworkAuditorControlStatus) error {
	_, err := c.doRequest(ctx, "PUT", "/api/v1/framework/"+frameworkID+"/auditor_control_status", status)
	return err
}

// SetFrameworkAuditorEvidenceStatus sets the auditor-visible evidence statuses.
func (c *AnecdotesClient) SetFrameworkAuditorEvidenceStatus(ctx context.Context, frameworkID string, status *FrameworkAuditorEvidenceStatus) error {
	_, err := c.doRequest(ctx, "PUT", "/api/v1/framework/"+frameworkID+"/auditor_evidence_status", status)
	return err
}

// UpdateFramework updates an existing framework
func (c *AnecdotesClient) UpdateFramework(ctx context.Context, frameworkID string, framework *FrameworkUpdateRequest) (*Framework, error) {
	respBody, err := c.doRequest(ctx, "PATCH", "/api/v1/framework/"+frameworkID, framework)
	if err != nil {
		return nil, err
	}

	// If response is empty, fetch the updated framework
	if len(respBody) == 0 {
		return c.GetFramework(ctx, frameworkID)
	}

	var result Framework
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Fallback: try fetching if parsing fails
		return c.GetFramework(ctx, frameworkID)
	}

	return &result, nil
}

// DeleteFramework deletes a framework
func (c *AnecdotesClient) DeleteFramework(ctx context.Context, frameworkID string) error {
	_, err := c.doRequest(ctx, "DELETE", "/api/v1/framework/"+frameworkID, nil)
	return err
}

// ListControlCategories retrieves all control categories
func (c *AnecdotesClient) ListControlCategories(ctx context.Context) ([]ControlCategory, error) {
	respBody, err := c.doRequest(ctx, "GET", "/api/v1/framework/category", nil)
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
func (c *AnecdotesClient) CreateControlCategory(ctx context.Context, category *ControlCategoryCreateRequest) (*ControlCategory, error) {
	respBody, err := c.doRequest(ctx, "POST", "/api/v1/framework/category", category)
	if err != nil {
		if IsServerError(err) {
			existing, lErr := c.getControlCategoryByName(ctx, category.CategoryName, category.FrameworkID)
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
func (c *AnecdotesClient) getControlCategoryByName(ctx context.Context, name, frameworkID string) (*ControlCategory, error) {
	categories, err := c.ListControlCategories(ctx)
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
func (c *AnecdotesClient) UpdateControlCategory(ctx context.Context, categoryID string, category *ControlCategoryUpdateRequest) error {
	_, err := c.doRequest(ctx, "PATCH", "/api/v1/framework/category/"+categoryID, category)
	return err
}

// DeleteControlCategory deletes a control category
func (c *AnecdotesClient) DeleteControlCategory(ctx context.Context, categoryID string) error {
	_, err := c.doRequest(ctx, "DELETE", "/api/v1/framework/category/"+categoryID, nil)
	return err
}

// GetControlCategory retrieves a specific category by searching through all categories
func (c *AnecdotesClient) GetControlCategory(ctx context.Context, categoryID string) (*ControlCategory, error) {
	categories, err := c.ListControlCategories(ctx)
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
func (c *AnecdotesClient) GetControl(ctx context.Context, frameworkID, controlID string) (*Control, error) {
	// Use POST /control/read to get control by ID
	body := map[string][]string{
		"controls_ids": {controlID},
	}
	respBody, err := c.doRequest(ctx, "POST", "/controls/control/read", body)
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
func (c *AnecdotesClient) ListControls(ctx context.Context, frameworkID string) ([]Control, error) {
	respBody, err := c.doRequest(ctx, "GET", "/controls/framework_controls?frameworks_ids="+frameworkID, nil)
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
func (c *AnecdotesClient) AddControl(ctx context.Context, frameworkID string, control *ControlCreateRequest) (*Control, error) {
	respBody, err := c.doRequest(ctx, "POST", "/controls/control?control_framework="+frameworkID, control)
	if err != nil {
		return nil, err
	}

	// The API may return just the control ID as a string
	var controlID string
	if err := json.Unmarshal(respBody, &controlID); err == nil && controlID != "" {
		// Fetch the full control details
		return c.GetControl(ctx, frameworkID, controlID)
	}

	// Try parsing as full Control object
	var result Control
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create control response: %w", err)
	}

	return &result, nil
}

// UpdateControl updates an existing control
func (c *AnecdotesClient) UpdateControl(ctx context.Context, frameworkID, controlID string, control *ControlUpdateRequest) (*Control, error) {
	respBody, err := c.doRequest(ctx, "PUT", "/controls/control/"+controlID, control)
	if err != nil {
		return nil, err
	}

	// Response may be empty on success, fetch updated control
	if len(respBody) == 0 || string(respBody) == "{}" {
		return c.GetControl(ctx, frameworkID, controlID)
	}

	var result Control
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If parsing fails, try to fetch the updated control
		return c.GetControl(ctx, frameworkID, controlID)
	}

	return &result, nil
}

// DeleteControl deletes a control from a framework
func (c *AnecdotesClient) DeleteControl(ctx context.Context, frameworkID, controlID string) error {
	_, err := c.doRequest(ctx, "DELETE", "/controls/control/"+controlID, nil)
	return err
}

// SetControlOwners replaces a control's owners. An empty slice clears them.
func (c *AnecdotesClient) SetControlOwners(ctx context.Context, controlID string, owners []string) error {
	if owners == nil {
		owners = []string{}
	}

	body := []map[string]interface{}{
		{
			"control_id":    controlID,
			"control_owner": owners,
		},
	}

	_, err := c.doRequest(ctx, "PATCH", "/controls/controls", body)
	return err
}

// SetControlMaturityLevel sets the maturity level for one or more controls
// An empty level clears the control's maturity level.
func (c *AnecdotesClient) SetControlMaturityLevel(ctx context.Context, controlID, maturityLevel string) error {
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

	_, err := c.doRequest(ctx, "PATCH", "/controls/controls/maturity_level", body)
	return err
}

// GetRequirement retrieves a specific requirement by ID
func (c *AnecdotesClient) GetRequirement(ctx context.Context, requirementID string) (*Requirement, error) {
	respBody, err := c.doRequest(ctx, "GET", "/api/v1/requirement/"+requirementID, nil)
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
		c.resolveRequirementStatusNames(ctx, []*Requirement{&results[0]})
		return &results[0], nil
	}

	// Try parsing as single object
	var result Requirement
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse requirement response: %w", err)
	}

	c.resolveRequirementStatusNames(ctx, []*Requirement{&result})
	return &result, nil
}

// ListRequirements retrieves all requirements from the Requirements Hub.
// GET /api/v1/requirement (no id) returns all requirement instances per OpenAPI.
func (c *AnecdotesClient) ListRequirements(ctx context.Context) ([]Requirement, error) {
	respBody, err := c.doRequest(ctx, "GET", "/api/v1/requirement", nil)
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
	c.resolveRequirementStatusNames(ctx, ptrs)

	return requirements, nil
}

// CreateRequirement creates a new requirement in the Requirements Hub. The
// second return value reports whether the requirement was resolved by name
// after a server error rather than read back from the create call.
func (c *AnecdotesClient) CreateRequirement(ctx context.Context, requirement *RequirementCreateRequest) (*Requirement, bool, error) {
	respBody, err := c.doRequest(ctx, "POST", "/api/v1/requirement", requirement)
	if err != nil {
		// A conflict is returned as-is: adopting a requirement that predates this
		// create would take ownership of an object Terraform did not create.
		if IsServerError(err) {
			existing, lErr := c.getRequirementByName(ctx, requirement.RequirementDescription)
			if lErr == nil && existing != nil {
				return existing, true, nil
			}
		}
		return nil, false, err
	}

	var requirementID string
	if err := json.Unmarshal(respBody, &requirementID); err == nil && requirementID != "" {
		created, err := c.GetRequirement(ctx, requirementID)
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
func (c *AnecdotesClient) getRequirementByName(ctx context.Context, name string) (*Requirement, error) {
	requirements, err := c.ListRequirements(ctx)
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
func (c *AnecdotesClient) UpdateRequirement(ctx context.Context, requirementID string, requirement *RequirementUpdateRequest) (*Requirement, error) {
	wrapped := map[string]interface{}{"requirement": requirement}
	_, err := c.doRequest(ctx, "PATCH", "/api/v1/requirement/"+requirementID, wrapped)
	if err != nil {
		return nil, err
	}

	// Fetch updated requirement
	return c.GetRequirement(ctx, requirementID)
}

// CreateRequirementView creates a Requirement View scoped beneath a parent
// requirement. Unlike CreateRequirement, a server error is never resolved by
// looking the view up by name: multiple views may share the same view_name, so
// a by-name match could silently adopt the wrong view.
func (c *AnecdotesClient) CreateRequirementView(ctx context.Context, view *RequirementViewCreateRequest) (*Requirement, error) {
	respBody, err := c.doRequest(ctx, "POST", "/api/v1/requirement", view)
	if err != nil {
		return nil, err
	}

	var requirementID string
	if err := json.Unmarshal(respBody, &requirementID); err == nil && requirementID != "" {
		return c.GetRequirement(ctx, requirementID)
	}

	var result Requirement
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create requirement view response: %w", err)
	}
	return &result, nil
}

// UpdateRequirementView updates an existing Requirement View.
// The API requires the request body wrapped in {"requirement": {...}}.
func (c *AnecdotesClient) UpdateRequirementView(ctx context.Context, requirementID string, view *RequirementViewUpdateRequest) (*Requirement, error) {
	wrapped := map[string]interface{}{"requirement": view}
	_, err := c.doRequest(ctx, "PATCH", "/api/v1/requirement/"+requirementID, wrapped)
	if err != nil {
		return nil, err
	}

	return c.GetRequirement(ctx, requirementID)
}

// DeleteRequirement deletes a requirement. The response reports how many
// requirements were removed; zero means the requirement was either already gone
// or could not be removed, so which one is confirmed by reading it back rather
// than reporting a success that did not happen.
func (c *AnecdotesClient) DeleteRequirement(ctx context.Context, requirementID string) error {
	respBody, err := c.doRequest(ctx, "POST", "/api/v1/requirement/delete", []string{requirementID})
	if err != nil {
		return err
	}

	// A response that cannot be parsed is not evidence of anything, so it falls
	// through to the read-back rather than counting as success.
	var result struct {
		DeletedCount int `json:"deleted_count"`
	}
	if err := json.Unmarshal(respBody, &result); err == nil && result.DeletedCount > 0 {
		return nil
	}

	_, err = c.GetRequirement(ctx, requirementID)
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not confirm requirement %s was deleted: %w", requirementID, err)
	}
	return fmt.Errorf("requirement %s still exists after the delete call: it may be a requirement "+
		"provided by the Anecdotes platform, which cannot be deleted (only custom requirements and "+
		"views can). Remove it from state with `terraform state rm` if Terraform should stop managing it",
		requirementID)
}

// ListRequirementStatuses fetches the available requirement status options.
// API: GET /api/v1/requirement/status
// Returns a map of status name → status ID.
func (c *AnecdotesClient) ListRequirementStatuses(ctx context.Context) (map[string]string, error) {
	respBody, err := c.doRequest(ctx, "GET", "/api/v1/requirement/status", nil)
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
func (c *AnecdotesClient) requirementStatusNameByID(ctx context.Context) map[string]string {
	statusMap, err := c.ListRequirementStatuses(ctx)
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
func (c *AnecdotesClient) resolveRequirementStatusNames(ctx context.Context, reqs []*Requirement) {
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

	nameByID := c.requirementStatusNameByID(ctx)
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
func (c *AnecdotesClient) LinkRequirementToControl(ctx context.Context, controlID, requirementID string) (*ControlRequirementLink, error) {
	defer c.parentLocks.lock("control:" + controlID)()

	// Read current control to get existing requirement IDs
	control, err := c.GetControl(ctx, "", controlID)
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
	if err := c.UpdateControlRequirements(ctx, controlID, updatedIDs); err != nil {
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
func (c *AnecdotesClient) UnlinkRequirementFromControl(ctx context.Context, controlID, requirementID string) error {
	defer c.parentLocks.lock("control:" + controlID)()

	// Read current control to get existing requirement IDs
	control, err := c.GetControl(ctx, "", controlID)
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

	return c.UpdateControlRequirements(ctx, controlID, updatedIDs)
}

// GetControlRequirementLink verifies a link exists between a control and requirement.
// Uses POST /controls/control/read to read control_requirement_ids from the control.
func (c *AnecdotesClient) GetControlRequirementLink(ctx context.Context, controlID, requirementID string) (*ControlRequirementLink, error) {
	control, err := c.GetControl(ctx, "", controlID)
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
func (c *AnecdotesClient) UpdateControlRequirements(ctx context.Context, controlID string, requirementIDs []string) error {
	payload := []map[string]interface{}{
		{
			"control_id":                   controlID,
			"control_related_requirements": requirementIDs,
			"requirements_recommendations": []string{},
		},
	}

	_, err := c.doRequest(ctx, "PATCH", "/controls/controls", payload)
	if err != nil {
		return fmt.Errorf("failed to update control requirements: %w", err)
	}

	return nil
}

// CreateFolder creates a new folder for organizing frameworks
func (c *AnecdotesClient) CreateFolder(ctx context.Context, folder *FolderCreateRequest) (*Folder, error) {
	respBody, err := c.doRequest(ctx, "POST", "/frameworks/v1/folders", folder)
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
func (c *AnecdotesClient) GetFolder(ctx context.Context, folderID string) (*Folder, error) {
	respBody, err := c.doRequest(ctx, "GET", "/frameworks/v1/folders/"+folderID, nil)
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
func (c *AnecdotesClient) ListFolders(ctx context.Context) ([]Folder, error) {
	respBody, err := c.doRequest(ctx, "GET", "/frameworks/v1/folders", nil)
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
func (c *AnecdotesClient) UpdateFolder(ctx context.Context, folderID string, folder *FolderUpdateRequest) (*Folder, error) {
	respBody, err := c.doRequest(ctx, "PATCH", "/frameworks/v1/folders/"+folderID, folder)
	if err != nil {
		return nil, err
	}

	// If response is empty, fetch the updated folder
	if len(respBody) == 0 {
		return c.GetFolder(ctx, folderID)
	}

	var result Folder
	if err := json.Unmarshal(respBody, &result); err != nil {
		return c.GetFolder(ctx, folderID)
	}

	return &result, nil
}

// DeleteFolder deletes a folder
func (c *AnecdotesClient) DeleteFolder(ctx context.Context, folderID string) error {
	_, err := c.doRequest(ctx, "DELETE", "/frameworks/v1/folders/"+folderID, nil)
	return err
}

// FindFrameworkFolder returns the ID of the folder containing the framework,
// or "" when the framework is not in any folder.
func (c *AnecdotesClient) FindFrameworkFolder(ctx context.Context, frameworkID string) (string, error) {
	folders, err := c.ListFolders(ctx)
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
func (c *AnecdotesClient) MoveFrameworkFolder(ctx context.Context, frameworkID, fromFolderID, toFolderID string) error {
	body := map[string]string{
		"remove_from_folder_id": fromFolderID,
		"add_to_folder_id":      toFolderID,
	}
	_, err := c.doRequest(ctx, "PUT", "/frameworks/v1/folders/framework/"+frameworkID, body)
	return err
}

// ListEvidences returns all evidences in the account.
// GET /evidence/v1/evidence returns a flat JSON array of Evidence objects.
func (c *AnecdotesClient) ListEvidences(ctx context.Context) ([]Evidence, error) {
	respBody, err := c.doRequest(ctx, "GET", "/evidence/v1/evidence", nil)
	if err != nil {
		return nil, err
	}

	var evidences []Evidence
	if err := json.Unmarshal(respBody, &evidences); err != nil {
		return nil, fmt.Errorf("failed to parse evidences list response: %w", err)
	}

	return evidences, nil
}

// LinkEvidenceToRequirement adds an evidence ID to a requirement's evidence list.
// Uses read-modify-write: reads current list, appends if missing, PATCHes back.
// The API uses `requirement_related_evidences` for write and `requirement_evidence_ids` for read.
func (c *AnecdotesClient) LinkEvidenceToRequirement(ctx context.Context, requirementID, evidenceID string) error {
	defer c.parentLocks.lock("requirement:" + requirementID)()

	req, err := c.GetRequirement(ctx, requirementID)
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

	_, err = c.UpdateRequirement(ctx, requirementID, updateReq)
	return err
}

// UnlinkEvidenceFromRequirement removes an evidence ID from a requirement's evidence list.
func (c *AnecdotesClient) UnlinkEvidenceFromRequirement(ctx context.Context, requirementID, evidenceID string) error {
	defer c.parentLocks.lock("requirement:" + requirementID)()

	req, err := c.GetRequirement(ctx, requirementID)
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

	_, err = c.UpdateRequirement(ctx, requirementID, updateReq)
	return err
}

// GetRequirementEvidenceLink checks if an evidence ID is linked to a requirement.
// Returns nil if linked, error if not found.
func (c *AnecdotesClient) GetRequirementEvidenceLink(ctx context.Context, requirementID, evidenceID string) error {
	req, err := c.GetRequirement(ctx, requirementID)
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

// GetControlMaturityLevel reads the maturity level from a control.
// Note: SetControlMaturityLevel already exists above (line ~449).
func (c *AnecdotesClient) GetControlMaturityLevel(ctx context.Context, controlID string) (string, error) {
	respBody, err := c.doRequest(ctx, "POST", "/controls/control/read", map[string][]string{
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
