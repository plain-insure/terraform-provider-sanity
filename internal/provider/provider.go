package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tessellator/go-sanity/sanity"
	"golang.org/x/oauth2"
)

var _ provider.Provider = &SanityProvider{}
var _ provider.ProviderWithMetadata = &SanityProvider{}

// SanityProvider defines the provider implementation.
type SanityProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and run locally, and "test" when running acceptance
	// testing.
	version string
}

// SanityProviderModel describes the provider data model.
type SanityProviderModel struct {
	Token types.String `tfsdk:"token"`
}

func (p *SanityProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sanity"
	resp.Version = p.version
}

func (p *SanityProvider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"token": {
				MarkdownDescription: "The auth token used to authenticate with Sanity. May be sourced from the `SANITY_TOKEN` environment variable instead of via this attribute.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Type:                types.StringType,
			},
		},
	}, nil
}

func (p *SanityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config SanityProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var token string
	if config.Token.Unknown {
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as token",
		)
		return
	}

	if config.Token.Null {
		token = os.Getenv("SANITY_TOKEN")
	} else {
		token = config.Token.Value
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Unable to find token",
			"Token cannot be an empty string",
		)
		return
	}

	tokenSrc := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), tokenSrc)

	client := sanity.NewClient(httpClient)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *SanityProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewCORSOriginResource,
		NewDatasetResource,
		NewProjectTokenResource,
	}
}

func (p *SanityProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewProjectDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &SanityProvider{
			version: version,
		}
	}
}
