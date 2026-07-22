// Copyright (c) Anecdotes AI
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/anecdotes-ai/terraform-provider-anecdotes/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FrameworkFolderDataSource{}

func NewFrameworkFolderDataSource() datasource.DataSource {
	return &FrameworkFolderDataSource{}
}

// FrameworkFolderDataSource defines the data source implementation.
type FrameworkFolderDataSource struct {
	client *client.AnecdotesClient
}

// FrameworkFolderDataSourceModel describes the data source data model.
type FrameworkFolderDataSourceModel struct {
	FolderID       types.String `tfsdk:"folder_id"`
	Name           types.String `tfsdk:"name"`
	FrameworksList types.List   `tfsdk:"frameworks_list"`
}

func (d *FrameworkFolderDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_framework_folder"
}

func (d *FrameworkFolderDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to look up an existing folder by name.",

		Attributes: map[string]schema.Attribute{
			"folder_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique identifier for the folder.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the folder to look up.",
			},
			"frameworks_list": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of framework IDs in this folder.",
			},
		},
	}
}

func (d *FrameworkFolderDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.AnecdotesClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.AnecdotesClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *FrameworkFolderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FrameworkFolderDataSourceModel

	// Read Terraform configuration data into the model
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// List all folders
	folders, err := d.client.ListFolders()
	if err != nil {
		addClientError(&resp.Diagnostics, "list folders", err)
		return
	}

	// Find the folder by name
	folderName := config.Name.ValueString()
	var foundFolder *client.Folder
	for _, f := range folders {
		if f.Name == folderName {
			foundFolder = &f
			break
		}
	}

	if foundFolder == nil {
		resp.Diagnostics.AddError(
			"Folder Not Found",
			fmt.Sprintf("No folder found with name: %s", folderName),
		)
		return
	}

	// Set state
	config.FolderID = types.StringValue(foundFolder.ID)
	config.Name = types.StringValue(foundFolder.Name)

	// Frameworks list
	if len(foundFolder.FrameworksList) > 0 {
		frameworksList, listDiags := types.ListValueFrom(ctx, types.StringType, foundFolder.FrameworksList)
		resp.Diagnostics.Append(listDiags...)
		config.FrameworksList = frameworksList
	} else {
		config.FrameworksList = types.ListNull(types.StringType)
	}

	diags = resp.State.Set(ctx, &config)
	resp.Diagnostics.Append(diags...)
}
