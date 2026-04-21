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

var _ datasource.DataSource = (*organizationSecretsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*organizationSecretsDataSource)(nil)

func NewOrganizationSecretsDataSource() datasource.DataSource {
	return &organizationSecretsDataSource{}
}

type organizationSecretsDataSource struct {
	client *crowciClient
}

type organizationSecretsDataSourceModel struct {
	OrgID   types.Int64                   `tfsdk:"org_id"`
	Secrets []organizationSecretItemModel `tfsdk:"secrets"`
}

type organizationSecretItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	RepoID    types.Int64  `tfsdk:"repo_id"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *organizationSecretsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_secrets"
}

func (d *organizationSecretsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
		"repo_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Repo scope of the secret.",
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
		Description: "Fetches all secrets for a specific organization on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the organization to list secrets for.",
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

func (d *organizationSecretsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationSecretsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var all []globalSecretAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/orgs/%d/secrets?page=%d&perPage=50", d.client.Host, data.OrgID.ValueInt64(), page)
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
				fmt.Sprintf("GET /orgs/%d/secrets returned status %d", data.OrgID.ValueInt64(), httpResp.StatusCode),
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

	secrets := make([]organizationSecretItemModel, len(all))
	for i, s := range all {
		secrets[i] = organizationSecretItemModel{
			ID:        types.Int64Value(s.ID),
			Name:      types.StringValue(s.Name),
			RepoID:    int64NullIfZero(s.RepoID),
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
