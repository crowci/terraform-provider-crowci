package provider

import (
	"context"
	"net/http"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*crowciProvider)(nil)

func New() func() provider.Provider {
	return func() provider.Provider {
		return &crowciProvider{}
	}
}

type crowciProvider struct{}

type crowciProviderModel struct {
	Host  types.String `tfsdk:"host"`
	Token types.String `tfsdk:"token"`
}

// crowciClient holds a configured HTTP client and the API base URL.
type crowciClient struct {
	Host       string
	HTTPClient *http.Client
}

func (p *crowciProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Crow CI Server.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "The base URL of the Crow CI API (e.g. https://ci.example.com). Falls back to CROWCI_HOST.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Personal access token used to authenticate against the Crow CI API. Falls back to CROWCI_TOKEN.",
			},
		},
	}
}

func (p *crowciProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config crowciProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := config.Token.ValueString()
	if token == "" {
		token = os.Getenv("CROWCI_TOKEN")
	}

	host := config.Host.ValueString()
	if host == "" {
		host = os.Getenv("CROWCI_HOST")
	}

	if token == "" {
		resp.Diagnostics.AddError(
			"Missing token",
			"Set the token attribute or the CROWCI_TOKEN environment variable.",
		)
		return
	}

	client := &crowciClient{
		Host: host,
		HTTPClient: &http.Client{
			Transport: &bearerTransport{
				token: token,
				base:  http.DefaultTransport,
			},
		},
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *crowciProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "crowci"
}

func (p *crowciProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserAccessTokenDataSource,
		NewUserAccessTokensDataSource,
		NewForgesDataSource,
		NewGlobalSecretDataSource,
		NewGlobalSecretsDataSource,
		NewOrganizationDataSource,
		NewOrganizationsDataSource,
	}
}

func (p *crowciProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserAccessTokenResource,
		NewGlobalSecretResource,
	}
}

// bearerTransport injects the Authorization: Bearer <token> header on every request.
type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}
