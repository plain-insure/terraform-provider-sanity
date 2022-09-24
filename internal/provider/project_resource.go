package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tessellator/go-sanity/sanity"
	"github.com/tessellator/terraform-provider-sanity/internal/provider/attribute_plan_modifier"
)

var _ resource.Resource = &ProjectResource{}
var _ resource.ResourceWithImportState = &ProjectResource{}

func NewProjectResource() resource.Resource {
	return &ProjectResource{}
}

type ProjectResource struct {
	client *sanity.Client
}

// ProjectResourceModel describes the resource data model.
type ProjectResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Organization        types.String `tfsdk:"organization"`
	StudioHost          types.String `tfsdk:"studio_host"`
	ExternalStudioHost  types.String `tfsdk:"external_studio_host"`
	Color               types.String `tfsdk:"color"`
	IsDisabledByUser    types.Bool   `tfsdk:"disabled_by_user"`
	ActivityFeedEnabled types.Bool   `tfsdk:"activity_feed_enabled"`
}

func (r *ProjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Provides a Sanity project. A project is the base resource for creating content, and the project may contain datasets, CORS origins, and tags.",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Computed:            true,
				MarkdownDescription: "The project ID, which you can find at the top of the project page in Sanity.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"name": {
				MarkdownDescription: "The project name.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"organization": {
				MarkdownDescription: "The name of the organization that owns the project.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"studio_host": {
				MarkdownDescription: "The studio host URL. This attribute exhibits two unique behaviors that are important to note. First, once the studio host URL is set, it may not be changed. Changing this value will force a replacement. Second, when the studio host is set, Sanity will automatically create a CORS entry for the studio host URL. This means that it is not necessary for you to create a CORS entry, and you will get a conflict error if you do.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
					resource.RequiresReplace(),
				},
			},
			"external_studio_host": {
				MarkdownDescription: "The external studio host URL.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"color": {
				MarkdownDescription: "The hex value for the project color. This is the color of the project icon at https://sanity.io/manage.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"disabled_by_user": {
				MarkdownDescription: "Indicates whether the project is archived. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Type:                types.BoolType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.Bool{Value: false}),
				},
			},
			"activity_feed_enabled": {
				MarkdownDescription: "Indicates whether the [activity feed](https://www.sanity.io/docs/activity-feed) is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Type:                types.BoolType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.Bool{Value: true}),
				},
			},
		},
	}, nil
}

func (r *ProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sanity.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ProjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.client.Projects.Create(ctx, &sanity.CreateProjectRequest{
		DisplayName:    data.Name.Value,
		OrganizationId: data.Organization.Value,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	entries, err := r.client.Projects.ListCORSEntries(ctx, project.Id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	for _, entry := range entries {
		_, err = r.client.Projects.DeleteCORSEntry(ctx, project.Id, entry.Id)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			r.client.Projects.Delete(ctx, project.Id)
			return
		}
	}

	requiresUpdate := !data.StudioHost.Null ||
		!data.ExternalStudioHost.Null ||
		!data.Color.Null ||
		!data.IsDisabledByUser.Null ||
		!data.ActivityFeedEnabled.Null

	if requiresUpdate {
		updateReq := &sanity.UpdateProjectRequest{}
		if !data.StudioHost.Null {
			updateReq.StudioHost = data.StudioHost.Value
		}
		if !data.ExternalStudioHost.Null {
			updateReq.ExternalStudioHost = data.ExternalStudioHost.Value
		}
		if !data.Color.Null {
			updateReq.Color = data.Color.Value
		}
		if !data.IsDisabledByUser.Null {
			updateReq.IsDisabledByUser = sanity.NewBool(data.IsDisabledByUser.Value)
		}
		if !data.ActivityFeedEnabled.Null {
			updateReq.ActivityFeedEnabled = sanity.NewBool(data.ActivityFeedEnabled.Value)
		}
		project, err = r.client.Projects.Update(ctx, project.Id, updateReq)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			r.client.Projects.Delete(ctx, project.Id)
			return
		}
	}

	data.Id = types.String{Value: project.Id}
	data.Name = types.String{Value: project.DisplayName}
	data.Organization = types.String{Value: project.OrganizationId}
	data.StudioHost = types.String{Value: project.StudioHost}
	data.ExternalStudioHost = types.String{Value: project.Metadata["externalStudioHost"]}
	data.Color = types.String{Value: project.Metadata["color"]}
	data.IsDisabledByUser = types.Bool{Value: project.IsDisabledByUser}
	data.ActivityFeedEnabled = types.Bool{Value: project.ActivityFeedEnabled}

	tflog.Trace(ctx, "created a sanity project", map[string]interface{}{"id": project.Id, "name": project.DisplayName})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ProjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null {
		resp.Diagnostics.AddError("Project id is null", "Project id is null")
		return
	}

	project, err := r.client.Projects.Get(ctx, data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Id = types.String{Value: project.Id}
	data.Name = types.String{Value: project.DisplayName}
	data.Organization = types.String{Value: project.OrganizationId}
	data.StudioHost = types.String{Value: project.StudioHost}
	data.ExternalStudioHost = types.String{Value: project.Metadata["externalStudioHost"]}
	data.Color = types.String{Value: project.Metadata["color"]}
	data.IsDisabledByUser = types.Bool{Value: project.IsDisabledByUser}
	data.ActivityFeedEnabled = types.Bool{Value: project.ActivityFeedEnabled}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *ProjectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null {
		resp.Diagnostics.AddError("Project id is null", "Project id is null")
		return
	}

	var studioHost string
	req.State.GetAttribute(ctx, path.Root("studio_host"), &studioHost)

	requiresUpdate := !data.Name.Null ||
		(!data.StudioHost.Null && studioHost == "") ||
		!data.ExternalStudioHost.Null ||
		!data.Color.Null ||
		!data.IsDisabledByUser.Null ||
		!data.ActivityFeedEnabled.Null

	if !requiresUpdate {
		return
	}

	updateReq := &sanity.UpdateProjectRequest{}
	if !data.Name.Null {
		updateReq.DisplayName = data.Name.Value
	}
	if studioHost == "" && !data.StudioHost.Null {
		updateReq.StudioHost = data.StudioHost.Value
	}
	if !data.ExternalStudioHost.Null {
		updateReq.ExternalStudioHost = data.ExternalStudioHost.Value
	}
	if !data.Color.Null {
		updateReq.Color = data.Color.Value
	}
	if !data.IsDisabledByUser.Null {
		updateReq.IsDisabledByUser = sanity.NewBool(data.IsDisabledByUser.Value)
	}
	if !data.ActivityFeedEnabled.Null {
		updateReq.ActivityFeedEnabled = sanity.NewBool(data.ActivityFeedEnabled.Value)
	}
	project, err := r.client.Projects.Update(ctx, data.Id.Value, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		r.client.Projects.Delete(ctx, project.Id)
		return
	}

	data.Id = types.String{Value: project.Id}
	data.Name = types.String{Value: project.DisplayName}
	data.Organization = types.String{Value: project.OrganizationId}
	data.StudioHost = types.String{Value: project.StudioHost}
	data.ExternalStudioHost = types.String{Value: project.Metadata["externalStudioHost"]}
	data.Color = types.String{Value: project.Metadata["color"]}
	data.IsDisabledByUser = types.Bool{Value: project.IsDisabledByUser}
	data.ActivityFeedEnabled = types.Bool{Value: project.ActivityFeedEnabled}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ProjectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null {
		resp.Diagnostics.AddError("Project id is null", "Project id is null")
		return
	}

	_, err := r.client.Projects.Delete(ctx, data.Id.Value)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("project %s could not be deleted, got error: %s", data.Id.Value, err))
		return
	}
}

func (r *ProjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
