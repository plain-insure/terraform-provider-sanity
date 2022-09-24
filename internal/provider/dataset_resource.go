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
)

var _ resource.Resource = &DatasetResource{}
var _ resource.ResourceWithImportState = &DatasetResource{}

func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

type DatasetResource struct {
	client *sanity.Client
}

type DatasetResourceModel struct {
	Project types.String `tfsdk:"project"`
	Name    types.String `tfsdk:"name"`
	AclMode types.String `tfsdk:"acl_mode"`
}

func (r *DatasetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *DatasetResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Provides a dataset to a Sanity project. A dataset is like a database for your content, and you manage its contents with a studio and query it with GROQ or GraphQL.",

		Attributes: map[string]tfsdk.Attribute{
			"project": {
				Required:            true,
				Type:                types.StringType,
				MarkdownDescription: "The ID of the project that the dataset belongs to.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"name": {
				Required:            true,
				Type:                types.StringType,
				MarkdownDescription: "The name of the dataset.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"acl_mode": {
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				MarkdownDescription: "The ACL mode for the data. Valid options are `public` and `private`.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
		},
	}, nil
}

func (r *DatasetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *DatasetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.Projects.CreateDataset(ctx, data.Project.Value, &sanity.CreateDatasetRequest{
		Name:    data.Name.Value,
		AclMode: data.AclMode.Value,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *DatasetResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	projectId := data.Project.Value

	datasets, err := r.client.Projects.ListDatasets(ctx, projectId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	var dataset sanity.Dataset
	found := false

	for _, d := range datasets {
		if d.Name == data.Name.Value {
			dataset = d
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError("dataset not found", "dataset not found")
		return
	}

	data.AclMode = types.String{Value: dataset.AclMode}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Provider Error", "Update is not supported on dataset")
}

func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *DatasetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.Null {
		resp.Diagnostics.AddError("Name is null", "Name is null")
		return
	}
	if data.Project.Null {
		resp.Diagnostics.AddError("Project is null", "Project is null")
		return
	}

	_, err := r.client.Projects.DeleteDataset(ctx, data.Project.Value, data.Name.Value)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("dataset %s could not be deleted, got error: %s", data.Name.Value, err))
		return
	}
}

func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Input Error", "The import identifier for a dataset should be in the form project-id/dataset-name")
		return
	}

	reqProject := resource.ImportStateRequest{ID: parts[0]}
	reqName := resource.ImportStateRequest{ID: parts[1]}

	resource.ImportStatePassthroughID(ctx, path.Root("project"), reqProject, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("name"), reqName, resp)
}
