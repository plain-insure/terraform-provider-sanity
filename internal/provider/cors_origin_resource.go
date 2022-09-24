package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tessellator/go-sanity/sanity"
	"github.com/tessellator/terraform-provider-sanity/internal/provider/attribute_plan_modifier"
)

var _ resource.Resource = &CORSOriginResource{}
var _ resource.ResourceWithImportState = &CORSOriginResource{}

func NewCORSOriginResource() resource.Resource {
	return &CORSOriginResource{}
}

type CORSOriginResource struct {
	client *sanity.Client
}

type CORSOriginResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Origin           types.String `tfsdk:"origin"`
	AllowCredentials types.Bool   `tfsdk:"allow_credentials"`
	Project          types.String `tfsdk:"project"`
}

func (r *CORSOriginResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cors_origin"
}

func (r *CORSOriginResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Provides a CORS origin to a Sanity project. A CORS origin is a host that can connect to the Sanity Project API.",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Computed:            true,
				MarkdownDescription: "The unique ID for the CORS origin.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"origin": {
				Required:            true,
				MarkdownDescription: "The origin you want to allow traffic from, stating explicitly the protocol, host name and port. Wildcards (`*`) are allowed. Use the following format: `protocol://host:port`.",
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"allow_credentials": {
				Optional: true,
				Computed: true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
					attribute_plan_modifier.DefaultValue(types.Bool{Value: true}),
				},
				MarkdownDescription: "Indicates whether the origin is allowed to send credentials (e.g. a session cookie or an authorization token). Defaults to `true`.",
				Type:                types.BoolType,
			},
			"project": {
				Required:            true,
				MarkdownDescription: "The ID of the project that the CORS origin belongs to.",
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (r *CORSOriginResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CORSOriginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *CORSOriginResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	corsReq := &sanity.CreateCORSEntryRequest{
		Origin:           data.Origin.Value,
		AllowCredentials: sanity.NewBool(true),
	}
	if !data.AllowCredentials.IsNull() {
		corsReq.AllowCredentials = sanity.NewBool(data.AllowCredentials.Value)
	}
	entry, err := r.client.Projects.CreateCORSEntry(ctx, data.Project.Value, corsReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Id = types.String{Value: fmt.Sprintf("%d", entry.Id)}
	data.AllowCredentials = types.Bool{Value: entry.AllowCredentials}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CORSOriginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *CORSOriginResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null {
		resp.Diagnostics.AddError("Entry id is null", "Entry id is null")
		return
	}
	if data.Project.Null {
		resp.Diagnostics.AddError("Project is null", "Project is null")
		return
	}

	entries, err := r.client.Projects.ListCORSEntries(ctx, data.Project.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	rawId := int64(0)
	_, err = fmt.Sscanf(data.Id.Value, "%d", &rawId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	var entry sanity.CORSEntry
	found := false
	for _, e := range entries {
		if e.Id == rawId {
			entry = e
			found = true
			break
		}
	}
	if !found {
		resp.Diagnostics.AddError("cors entry not found", "cors entry not found")
		return
	}

	data.AllowCredentials = types.Bool{Value: entry.AllowCredentials}
	data.Origin = types.String{Value: entry.Origin}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CORSOriginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error", "Update is not supported on CORS entry")
}

func (r *CORSOriginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *CORSOriginResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null {
		resp.Diagnostics.AddError("Entry id is null", "Entry id is null")
		return
	}
	if data.Project.Null {
		resp.Diagnostics.AddError("Project is null", "Project is null")
		return
	}

	rawId := int64(0)
	_, err := fmt.Sscanf(data.Id.Value, "%d", &rawId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	_, err = r.client.Projects.DeleteCORSEntry(ctx, data.Project.Value, rawId)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("entry %s could not be deleted, got error: %s", data.Id.Value, err))
		return
	}
}

func (r *CORSOriginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	projectId, origin, _ := strings.Cut(req.ID, "/")
	if projectId == "" || origin == "" {
		resp.Diagnostics.AddError("Import Error", "The format for importing a CORS origin is project-id/origin")
		return
	}

	entries, err := r.client.Projects.ListCORSEntries(ctx, projectId)
	if err != nil {
		resp.Diagnostics.AddError("Import Error", err.Error())
		return
	}

	for _, e := range entries {
		if e.Origin == origin {
			resource.ImportStatePassthroughID(ctx, path.Root("id"), resource.ImportStateRequest{ID: fmt.Sprintf("%d", e.Id)}, resp)
			resource.ImportStatePassthroughID(ctx, path.Root("project"), resource.ImportStateRequest{ID: projectId}, resp)
			return
		}
	}

	resp.Diagnostics.AddError("Import Error", "The requested CORS origin was not found")
}
