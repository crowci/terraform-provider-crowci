package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*repositorySecretsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositorySecretsDataSource)(nil)

func NewRepositorySecretsDataSource() datasource.DataSource {
	return &repositorySecretsDataSource{}
}

type repositorySecretsDataSource struct {
	client *crowciClient
}

type repositorySecretsDataSourceModel struct {
	RepoID  types.Int64                  `tfsdk:"repo_id"`
	Secrets []repositorySecretItemModel  `tfsdk:"secrets"`
}

type repositorySecretItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *repositorySecretsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_secrets"
}

func (d *repositorySecretsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	secretAttrs := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Secret ID.",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Secret name.",
		},
		"events": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Events that trigger the secret.",
		},
		"images": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Container images the secret is available to.",
		},
		"source": schema.StringAttribute{
			Computed:    true,
			Description: "Source of the secret.",
		},
		"created_at": schema.Int64Attribute{
			Computed:    true,
			Description: "Creation time as a Unix timestamp.",
		},
		"updated_at": schema.Int64Attribute{
			Computed:    true,
			Description: "Last update time as a Unix timestamp.",
		},
	}

	resp.Schema = schema.Schema{
		Description: "Fetches all secrets for a specific repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository to list secrets for.",
			},
			"secrets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: secretAttrs,
				},
			},
		},
	}
}

func (d *repositorySecretsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*crowciClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *crowciClient, got %T", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *repositorySecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositorySecretsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var all []globalSecretAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/repos/%d/secrets?page=%d&perPage=50", d.client.Host, data.RepoID.ValueInt64(), page)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			resp.Diagnostics.AddError("Failed to build request", err.Error())
			return
		}

		httpResp, err := d.client.HTTPClient.Do(httpReq)
		if err != nil {
			resp.Diagnostics.AddError("API request failed", err.Error())
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			resp.Diagnostics.AddError(
				"Unexpected API response",
				fmt.Sprintf("GET /repos/%d/secrets returned status %d", data.RepoID.ValueInt64(), httpResp.StatusCode),
			)
			return
		}

		var pageResults []globalSecretAPIResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&pageResults); err != nil {
			resp.Diagnostics.AddError("Failed to decode response", err.Error())
			return
		}

		all = append(all, pageResults...)
		if len(pageResults) < 50 {
			break
		}
	}

	secrets := make([]repositorySecretItemModel, len(all))
	for i, s := range all {
		secrets[i] = repositorySecretItemModel{
			ID:        types.Int64Value(s.ID),
			Name:      types.StringValue(s.Name),
			Source:    types.StringValue(s.Source),
			CreatedAt: types.Int64Value(s.CreatedAt),
			UpdatedAt: types.Int64Value(s.UpdatedAt),
			Events:    stringsToList(s.Events),
			Images:    stringsToList(s.Images),
		}
	}

	data.Secrets = secrets
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
