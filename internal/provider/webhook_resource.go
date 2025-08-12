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

// WebhookRuleModel describes the webhook rule data model.
type WebhookRuleModel struct {
	On         []types.String `tfsdk:"on"`
	Filter     types.String   `tfsdk:"filter"`
	Projection types.String   `tfsdk:"projection"`
}

// WebhookResourceModel describes the resource data model.
type WebhookResourceModel struct {
	Id               types.String            `tfsdk:"id"`
	ProjectId        types.String            `tfsdk:"project_id"`
	Type             types.String            `tfsdk:"type"`
	Name             types.String            `tfsdk:"name"`
	Dataset          types.String            `tfsdk:"dataset"`
	URL              types.String            `tfsdk:"url"`
	HttpMethod       types.String            `tfsdk:"http_method"`
	ApiVersion       types.String            `tfsdk:"api_version"`
	IncludeDrafts    types.Bool              `tfsdk:"include_drafts"`
	Headers          map[string]types.String `tfsdk:"headers"`
	Rule             *WebhookRuleModel       `tfsdk:"rule"`
	Secret           types.String            `tfsdk:"secret"`
	IsDisabledByUser types.Bool              `tfsdk:"is_disabled_by_user"`
	IsDisabled       types.Bool              `tfsdk:"is_disabled"`
	CreatedAt        types.String            `tfsdk:"created_at"`
	UpdatedAt        types.String            `tfsdk:"updated_at"`
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
			"type": {
				MarkdownDescription: "The type of webhook. Can be 'document' or 'transaction'.",
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
				MarkdownDescription: "The HTTP method used for webhook requests. Defaults to `POST`. Only available for document webhooks.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.String{Value: "POST"}),
				},
			},
			"api_version": {
				MarkdownDescription: "The API version used for webhook payloads. Defaults to the current API version. Only available for document webhooks.",
				Optional:            true,
				Computed:            true,
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"include_drafts": {
				MarkdownDescription: "Whether draft documents trigger webhook notifications. Defaults to `false`. Only available for document webhooks.",
				Optional:            true,
				Computed:            true,
				Type:                types.BoolType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.Bool{Value: false}),
				},
			},
			"headers": {
				MarkdownDescription: "Custom HTTP headers sent with webhook requests. Only available for document webhooks.",
				Optional:            true,
				Type: types.MapType{
					ElemType: types.StringType,
				},
			},
			"rule": {
				MarkdownDescription: "The rule configuration for the webhook. Only available for document webhooks.",
				Optional:            true,
				Attributes: tfsdk.SingleNestedAttributes(map[string]tfsdk.Attribute{
					"on": {
						MarkdownDescription: "The events that trigger the webhook. Can be 'create', 'update', or 'delete'.",
						Required:            true,
						Type: types.ListType{
							ElemType: types.StringType,
						},
					},
					"filter": {
						MarkdownDescription: "A GROQ filter expression to determine which documents trigger the webhook.",
						Optional:            true,
						Type:                types.StringType,
					},
					"projection": {
						MarkdownDescription: "A GROQ projection to determine what data to include in the webhook payload.",
						Optional:            true,
						Type:                types.StringType,
					},
				}),
			},
			"secret": {
				MarkdownDescription: "Secret used for webhook signature verification. Only available for document webhooks.",
				Optional:            true,
				Sensitive:           true,
				Type:                types.StringType,
			},
			"is_disabled_by_user": {
				MarkdownDescription: "Whether the webhook is disabled by the user. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Type:                types.BoolType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					attribute_plan_modifier.DefaultValue(types.Bool{Value: false}),
				},
			},
			"is_disabled": {
				MarkdownDescription: "Whether the webhook is currently disabled (read-only).",
				Computed:            true,
				Type:                types.BoolType,
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

	// Validate webhook type restrictions
	if data.Type.Value == "transaction" {
		// For transaction webhooks, only certain fields are allowed
		if !data.HttpMethod.Null || !data.ApiVersion.Null || !data.IncludeDrafts.Null ||
			len(data.Headers) > 0 || data.Rule != nil || !data.Secret.Null {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"For transaction webhooks, only type, name, url, dataset, and is_disabled_by_user can be set.",
			)
			return
		}
	}

	// Convert headers map from Terraform types to Go strings
	headers := make(map[string]string)
	for k, v := range data.Headers {
		headers[k] = v.Value
	}

	createReq := &sanity.CreateWebhookRequest{
		Type:    data.Type.Value,
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
	if data.Rule != nil {
		rule := &sanity.WebhookRule{}

		// Convert On array
		for _, on := range data.Rule.On {
			rule.On = append(rule.On, on.Value)
		}

		if !data.Rule.Filter.Null {
			rule.Filter = data.Rule.Filter.Value
		}
		if !data.Rule.Projection.Null {
			rule.Projection = data.Rule.Projection.Value
		}

		createReq.Rule = rule
	}
	if !data.Secret.Null {
		createReq.Secret = data.Secret.Value
	}
	if !data.IsDisabledByUser.Null {
		createReq.IsDisabledByUser = sanity.NewBool(data.IsDisabledByUser.Value)
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

	// Validate webhook type restrictions
	if data.Type.Value == "transaction" {
		// For transaction webhooks, only certain fields are allowed
		if !data.HttpMethod.Null || !data.ApiVersion.Null || !data.IncludeDrafts.Null ||
			len(data.Headers) > 0 || data.Rule != nil || !data.Secret.Null {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"For transaction webhooks, only type, name, url, dataset, and is_disabled_by_user can be set.",
			)
			return
		}
	}

	updateReq := &sanity.UpdateWebhookRequest{}
	requiresUpdate := false

	if !data.Type.Null {
		updateReq.Type = data.Type.Value
		requiresUpdate = true
	}
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
	if data.Rule != nil {
		rule := &sanity.WebhookRule{}

		// Convert On array
		for _, on := range data.Rule.On {
			rule.On = append(rule.On, on.Value)
		}

		if !data.Rule.Filter.Null {
			rule.Filter = data.Rule.Filter.Value
		}
		if !data.Rule.Projection.Null {
			rule.Projection = data.Rule.Projection.Value
		}

		updateReq.Rule = rule
		requiresUpdate = true
	}
	if !data.Secret.Null {
		updateReq.Secret = data.Secret.Value
		requiresUpdate = true
	}
	if !data.IsDisabledByUser.Null {
		updateReq.IsDisabledByUser = sanity.NewBool(data.IsDisabledByUser.Value)
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
	data.Type = types.String{Value: webhook.Type}
	data.Name = types.String{Value: webhook.Name}
	data.Dataset = types.String{Value: webhook.Dataset}
	data.URL = types.String{Value: webhook.URL}
	data.HttpMethod = types.String{Value: webhook.HttpMethod}
	data.ApiVersion = types.String{Value: webhook.ApiVersion}
	data.IncludeDrafts = types.Bool{Value: webhook.IncludeDrafts}
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

	// Convert rule structure
	if webhook.Rule != nil {
		if data.Rule == nil {
			data.Rule = &WebhookRuleModel{}
		}

		// Convert On array
		data.Rule.On = make([]types.String, len(webhook.Rule.On))
		for i, on := range webhook.Rule.On {
			data.Rule.On[i] = types.String{Value: on}
		}

		// For optional fields, only set them if they have a value from the API
		// This prevents changing null to empty string
		if webhook.Rule.Filter != "" {
			data.Rule.Filter = types.String{Value: webhook.Rule.Filter}
		} else {
			data.Rule.Filter = types.String{Null: true}
		}

		if webhook.Rule.Projection != "" {
			data.Rule.Projection = types.String{Value: webhook.Rule.Projection}
		} else {
			data.Rule.Projection = types.String{Null: true}
		}
	}

	// Preserve the secret from state since it's not returned by the API
	if webhook.Secret != "" {
		data.Secret = types.String{Value: webhook.Secret}
	}
}
