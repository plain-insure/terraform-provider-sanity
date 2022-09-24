package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tessellator/go-sanity/sanity"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &ProjectDataSource{}

func NewProjectDataSource() datasource.DataSource {
	return &ProjectDataSource{}
}

// ProjectDataSource defines the data source implementation.
type ProjectDataSource struct {
	client *sanity.Client
}

// ProjectDataSourceModel describes the data source data model.
type ProjectDataSourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Organization        types.String `tfsdk:"organization"`
	StudioHost          types.String `tfsdk:"studio_host"`
	ExternalStudioHost  types.String `tfsdk:"external_studio_host"`
	IsDisabledByUser    types.Bool   `tfsdk:"disabled_by_user"`
	ActivityFeedEnabled types.Bool   `tfsdk:"activity_feed_enabled"`
}

func (d *ProjectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *ProjectDataSource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Gets a Sanity project by its ID. A project is the base resource for creating content, and the project may contain datasets, CORS origins, and tags.",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "The project ID, which you can find at the top of the project page in Sanity.",
				Type:                types.StringType,
				Required:            true,
			},
			"name": {
				MarkdownDescription: "The project name.",
				Type:                types.StringType,
				Computed:            true,
			},
			"organization": {
				MarkdownDescription: "The name of the organization that owns the project.",
				Type:                types.StringType,
				Computed:            true,
			},
			"studio_host": {
				MarkdownDescription: "The studio host URL.",
				Type:                types.StringType,
				Computed:            true,
			},
			"external_studio_host": {
				MarkdownDescription: "The external studio host URL.",
				Type:                types.StringType,
				Computed:            true,
			},
			"disabled_by_user": {
				MarkdownDescription: "Indicates whether the project is archived.",
				Computed:            true,
				Type:                types.BoolType,
			},
			"activity_feed_enabled": {
				MarkdownDescription: "Indicates whether the [activity feed](https://www.sanity.io/docs/activity-feed) is enabled.",
				Computed:            true,
				Type:                types.BoolType,
			},
		},
	}, nil
}

func (d *ProjectDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sanity.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *sanity.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *ProjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null {
		resp.Diagnostics.AddError("Project id is null", "Project id is null")
		return
	}

	project, err := d.client.Projects.Get(ctx, data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Id = types.String{Value: project.Id}
	data.Name = types.String{Value: project.DisplayName}
	data.Organization = types.String{Value: project.OrganizationId}
	data.StudioHost = types.String{Value: project.StudioHost}
	data.ExternalStudioHost = types.String{Value: project.Metadata["externalStudioHost"]}
	data.IsDisabledByUser = types.Bool{Value: project.IsDisabledByUser}
	data.ActivityFeedEnabled = types.Bool{Value: project.ActivityFeedEnabled}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
