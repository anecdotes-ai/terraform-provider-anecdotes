// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure AnecdotesProvider satisfies various provider interfaces.
var _ provider.Provider = &AnecdotesProvider{}

// AnecdotesProvider defines the provider implementation.
type AnecdotesProvider struct {
	version string
}

// AnecdotesProviderModel describes the provider data model.
type AnecdotesProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
	APIURL types.String `tfsdk:"api_url"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AnecdotesProvider{
			version: version,
		}
	}
}

func (p *AnecdotesProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "anecdotes"
	resp.Version = p.version
}

func (p *AnecdotesProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing Anecdotes GRC platform resources.",
		MarkdownDescription: `
# Anecdotes Terraform Provider

The Anecdotes provider allows you to manage compliance frameworks, controls, and requirements 
in the Anecdotes GRC platform using Infrastructure as Code.

## Resource Model

` + "```" + `
anecdotes_framework          # Framework container (SOC2, ISO 27001, etc.)
    │
    └── anecdotes_control    # Controls within a framework (1:N)
            │
            └── anecdotes_mapping_control_requirement  # Links to requirements (M:N)
                    │
                    └── anecdotes_requirement  # Standalone requirements (shared)
` + "```" + `

## Key Concepts

- **Framework**: A compliance standard container (e.g., SOC2, ISO 27001)
- **Control**: A prescriptive statement of what should be implemented
- **Requirement**: An operational action that enforces controls (can be shared across frameworks)
- **Control-Requirement Link**: The M:N relationship enabling cross-mapping

## Authentication

Generate an API token from the Anecdotes platform:

1. Log into Anecdotes as an Admin user
2. Navigate to **Administration → API Tokens**
3. Create a new token with the **Admin** role
4. Copy the token and store it securely

The API key is exchanged for a JWT Bearer token (valid for 60 minutes, auto-refreshed).

> **Note:** The exchange happens when the provider is configured, so ` + "`terraform plan`" + `
> and ` + "`terraform apply`" + ` require network access to the Anecdotes API and valid
> credentials even when no changes are planned. ` + "`terraform validate`" + ` works offline.

Example usage is generated from ` + "`examples/provider/provider.tf`" + `.
`,
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description:         "The Anecdotes API key for authentication. Can also be set via ANECDOTES_API_KEY environment variable.",
				MarkdownDescription: "The Anecdotes API key for authentication. Can also be set via `ANECDOTES_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_url": schema.StringAttribute{
				Description:         "The Anecdotes API base URL. Must use https. Defaults to https://api.anecdotes.ai. Can also be set via ANECDOTES_API_URL environment variable.",
				MarkdownDescription: "The Anecdotes API base URL. Must use `https`. Defaults to `https://api.anecdotes.ai`. Can also be set via the `ANECDOTES_API_URL` environment variable.",
				Optional:            true,
				Validators: []validator.String{
					requireHTTPSURL(),
				},
			},
		},
	}
}

func (p *AnecdotesProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config AnecdotesProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get API key from config or environment variable
	apiKey := config.APIKey.ValueString()
	if config.APIKey.IsNull() || config.APIKey.IsUnknown() || apiKey == "" {
		apiKey = os.Getenv("ANECDOTES_API_KEY")
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The Anecdotes API key is required. Set it in the provider configuration "+
				"or via the ANECDOTES_API_KEY environment variable.",
		)
		return
	}

	// Get API base URL from config or environment variable, with default fallback
	apiURL := config.APIURL.ValueString()
	if config.APIURL.IsNull() || config.APIURL.IsUnknown() || apiURL == "" {
		apiURL = os.Getenv("ANECDOTES_API_URL")
	}
	if apiURL == "" {
		apiURL = "https://api.anecdotes.ai"
	}

	// The environment variable bypasses schema validation, so the resolved URL is
	// checked here as well.
	if summary, detail := checkAPIURL(apiURL); summary != "" {
		resp.Diagnostics.AddError(summary, detail)
		return
	}

	// Create API client
	apiClient, err := client.NewAnecdotesClient(apiKey, apiURL)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Anecdotes API Client",
			"An unexpected error occurred when creating the Anecdotes API client. "+
				"If the error is not clear, please contact Anecdotes support.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	// Make the client available to resources and data sources
	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func (p *AnecdotesProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewFrameworkFolderResource,
		NewFrameworkResource,
		NewControlCategoryResource,
		NewControlResource,
		NewMappingControlRequirementResource,
		NewRequirementResource,
		NewMappingRequirementEvidenceResource,
	}
}

func (p *AnecdotesProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFrameworkDataSource,
		NewFrameworksDataSource,
		NewControlDataSource,
		NewControlsDataSource,
		NewControlCategoryDataSource,
		NewControlCategoriesDataSource,
		NewRequirementDataSource,
		NewRequirementsDataSource,
		NewFrameworkFolderDataSource,
		NewFrameworkFoldersDataSource,
		NewEvidencesDataSource,
	}
}
