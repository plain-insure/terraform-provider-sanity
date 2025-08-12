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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tessellator/go-sanity/sanity"
	"github.com/tessellator/terraform-provider-sanity/internal/provider/attribute_plan_modifier"
)

var _ resource.Resource = &WebhookResource{}
var _ resource.ResourceWithImportState = &WebhookResource{}

func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

type WebhookResource struct {
	client *sanity.Client
}

// WebhookResourceModel describes the resource data model.
type WebhookResourceModel struct {
	Id             types.String            `tfsdk:"id"`
	ProjectId      types.String            `tfsdk:"project_id"`
	Name           types.String            `tfsdk:"name"`
	Dataset        types.String            `tfsdk:"dataset"`
	URL            types.String            `tfsdk:"url"`
	HttpMethod     types.String            `tfsdk:"http_method"`
	ApiVersion     types.String            `tfsdk:"api_version"`
	IncludeDrafts  types.Bool              `tfsdk:"include_drafts"`
	Headers        map[string]types.String `tfsdk:"headers"`
	Filter         types.String            `tfsdk:"filter"`
	Secret         types.String            `tfsdk:"secret"`
	IsDisabled     types.Bool              `tfsdk:"is_disabled"`
	CreatedAt      types.String            `tfsdk:"created_at"`
	UpdatedAt      types.String            `tfsdk:"updated_at"`
}

func (r *WebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Provides a Sanity webhook. Webhooks allow you to get notified when content is created, updated, or deleted in your Sanity project.",

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Computed:            true,
				MarkdownDescription: "The webhook ID.",
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
				Type: types.StringType,
			},
			"project_id": {
				MarkdownDescription: "The project ID that this webhook belongs to.",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"name": {
				MarkdownDescription: "The human-readable name for the webhook.",
				Required:            true,
				Type:                types.StringType,
			},
			"dataset": {
				MarkdownDescription: "The dataset this webhook is configured for.",
				Required:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"url": {
				MarkdownDescription: "The endpoint URL that will receive webhook notifications.",
				Required:            true,
				Type:                types.StringType,
			},
			"http_method": {
				MarkdownDescription: "The HTTP method used for webhook requests. Defaults to `POST`.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.String{Value: "POST"}),
				},
			},
			"api_version": {
				MarkdownDescription: "The API version used for webhook payloads. Defaults to the current API version.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"include_drafts": {
				MarkdownDescription: "Whether draft documents trigger webhook notifications. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Type:                types.BoolType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.Bool{Value: false}),
				},
			},
			"headers": {
				MarkdownDescription: "Custom HTTP headers sent with webhook requests.",
				Optional:            true,
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
			"filter": {
				MarkdownDescription: "A GROQ filter expression to determine which documents trigger the webhook.",
				Optional:            true,
				Type:                types.StringType,
			},
			"secret": {
				MarkdownDescription: "Secret used for webhook signature verification.",
				Optional:            true,
				Sensitive:           true,
				Type:                types.StringType,
			},
			"is_disabled": {
				MarkdownDescription: "Whether the webhook is currently disabled. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Type:                types.BoolType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.Bool{Value: false}),
				},
			},
			"created_at": {
				Computed:            true,
				MarkdownDescription: "The time the webhook was created.",
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"updated_at": {
				Computed:            true,
				MarkdownDescription: "The time the webhook was last updated.",
				Type:                types.StringType,
			},
		},
	}, nil
}

func (r *WebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sanity.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sanity.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert headers map from Terraform types to Go strings
	headers := make(map[string]string)
	for k, v := range data.Headers {
		headers[k] = v.Value
	}

	createReq := &sanity.CreateWebhookRequest{
		Name:    data.Name.Value,
		Dataset: data.Dataset.Value,
		URL:     data.URL.Value,
	}

	if !data.HttpMethod.Null {
		createReq.HttpMethod = data.HttpMethod.Value
	}
	if !data.ApiVersion.Null {
		createReq.ApiVersion = data.ApiVersion.Value
	}
	if !data.IncludeDrafts.Null {
		createReq.IncludeDrafts = sanity.NewBool(data.IncludeDrafts.Value)
	}
	if len(headers) > 0 {
		createReq.Headers = headers
	}
	if !data.Filter.Null {
		createReq.Filter = data.Filter.Value
	}
	if !data.Secret.Null {
		createReq.Secret = data.Secret.Value
	}

	webhook, err := r.client.Webhooks.Create(ctx, data.ProjectId.Value, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	r.updateModelFromWebhook(data, webhook)

	tflog.Trace(ctx, "created a sanity webhook", map[string]interface{}{"id": webhook.Id, "project_id": webhook.ProjectId})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *WebhookResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null || data.ProjectId.Null {
		resp.Diagnostics.AddError("Missing required values", "Webhook id and project_id are required")
		return
	}

	webhook, err := r.client.Webhooks.Get(ctx, data.ProjectId.Value, data.Id.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	r.updateModelFromWebhook(data, webhook)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *WebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null || data.ProjectId.Null {
		resp.Diagnostics.AddError("Missing required values", "Webhook id and project_id are required")
		return
	}

	updateReq := &sanity.UpdateWebhookRequest{}
	requiresUpdate := false

	if !data.Name.Null {
		updateReq.Name = data.Name.Value
		requiresUpdate = true
	}
	if !data.URL.Null {
		updateReq.URL = data.URL.Value
		requiresUpdate = true
	}
	if !data.HttpMethod.Null {
		updateReq.HttpMethod = data.HttpMethod.Value
		requiresUpdate = true
	}
	if !data.ApiVersion.Null {
		updateReq.ApiVersion = data.ApiVersion.Value
		requiresUpdate = true
	}
	if !data.IncludeDrafts.Null {
		updateReq.IncludeDrafts = sanity.NewBool(data.IncludeDrafts.Value)
		requiresUpdate = true
	}
	if len(data.Headers) > 0 {
		headers := make(map[string]string)
		for k, v := range data.Headers {
			headers[k] = v.Value
		}
		updateReq.Headers = headers
		requiresUpdate = true
	}
	if !data.Filter.Null {
		updateReq.Filter = data.Filter.Value
		requiresUpdate = true
	}
	if !data.Secret.Null {
		updateReq.Secret = data.Secret.Value
		requiresUpdate = true
	}
	if !data.IsDisabled.Null {
		updateReq.IsDisabled = sanity.NewBool(data.IsDisabled.Value)
		requiresUpdate = true
	}

	if !requiresUpdate {
		return
	}

	webhook, err := r.client.Webhooks.Update(ctx, data.ProjectId.Value, data.Id.Value, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	r.updateModelFromWebhook(data, webhook)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *WebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Id.Null || data.ProjectId.Null {
		resp.Diagnostics.AddError("Missing required values", "Webhook id and project_id are required")
		return
	}

	_, err := r.client.Webhooks.Delete(ctx, data.ProjectId.Value, data.Id.Value)

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("webhook %s could not be deleted, got error: %s", data.Id.Value, err))
		return
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Input Error", "The import identifier for a webhook should be in the form project-id/webhook-id")
		return
	}

	reqProjectId := resource.ImportStateRequest{ID: parts[0]}
	reqWebhookId := resource.ImportStateRequest{ID: parts[1]}

	resource.ImportStatePassthroughID(ctx, path.Root("project_id"), reqProjectId, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), reqWebhookId, resp)
}

// updateModelFromWebhook updates the Terraform model with data from the API webhook struct
func (r *WebhookResource) updateModelFromWebhook(data *WebhookResourceModel, webhook *sanity.Webhook) {
	data.Id = types.String{Value: webhook.Id}
	data.ProjectId = types.String{Value: webhook.ProjectId}
	data.Name = types.String{Value: webhook.Name}
	data.Dataset = types.String{Value: webhook.Dataset}
	data.URL = types.String{Value: webhook.URL}
	data.HttpMethod = types.String{Value: webhook.HttpMethod}
	data.ApiVersion = types.String{Value: webhook.ApiVersion}
	data.IncludeDrafts = types.Bool{Value: webhook.IncludeDrafts}
	data.Filter = types.String{Value: webhook.Filter}
	data.IsDisabled = types.Bool{Value: webhook.IsDisabled}
	data.CreatedAt = types.String{Value: webhook.CreatedAt.Format("2006-01-02T15:04:05Z")}
	data.UpdatedAt = types.String{Value: webhook.UpdatedAt.Format("2006-01-02T15:04:05Z")}

	// Convert headers map from Go strings to Terraform types
	if webhook.Headers != nil && len(webhook.Headers) > 0 {
		if data.Headers == nil {
			data.Headers = make(map[string]types.String)
		}
		for k, v := range webhook.Headers {
			data.Headers[k] = types.String{Value: v}
		}
	}

	// Preserve the secret from state since it's not returned by the API
	if webhook.Secret != "" {
		data.Secret = types.String{Value: webhook.Secret}
	}
}