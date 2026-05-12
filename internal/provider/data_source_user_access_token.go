package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*userAccessTokenDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*userAccessTokenDataSource)(nil)

func NewUserAccessTokenDataSource() datasource.DataSource {
	return &userAccessTokenDataSource{}
}

type userAccessTokenDataSource struct {
	datasourceWithClient
}

type userAccessTokenDataSourceModel struct {
	TokenID   types.Int64  `tfsdk:"token_id"`
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Scopes    types.List   `tfsdk:"scopes"`
	UserID    types.Int64  `tfsdk:"user_id"`
	OrgID     types.Int64  `tfsdk:"org_id"`
	RepoID    types.Int64  `tfsdk:"repo_id"`
	ExpiresAt types.Int64  `tfsdk:"expires_at"`
	LastUsed  types.Int64  `tfsdk:"last_used"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *userAccessTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_access_token"
}

func (d *userAccessTokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get an access token.",
		Attributes: map[string]schema.Attribute{
			"token_id": schema.Int64Attribute{
				Required:    true,
				Description: "The token's id.",
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Token ID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Token name.",
			},
			"scopes": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Token scopes.",
			},
			"user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user who owns this token.",
			},
			"org_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Org scope of the token.",
			},
			"repo_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Repo scope of the token.",
			},
			"expires_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Expiry as a Unix timestamp.",
			},
			"last_used": schema.Int64Attribute{
				Computed:    true,
				Description: "Last use time as a Unix timestamp.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation time as a Unix timestamp.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Last update time as a Unix timestamp.",
			},
		},
	}
}

func (d *userAccessTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data userAccessTokenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/user/access-tokens/%d", d.client.Host, data.TokenID.ValueInt64())
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
			fmt.Sprintf("GET /user/access-tokens/%d returned status %d", data.TokenID.ValueInt64(), httpResp.StatusCode),
		)
		return
	}

	var result accessTokenAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	data.ID = types.Int64Value(result.ID)
	data.Name = types.StringValue(result.Name)
	data.UserID = types.Int64Value(result.UserID)
	data.OrgID = types.Int64Value(result.OrgID)
	data.RepoID = types.Int64Value(result.RepoID)
	data.ExpiresAt = types.Int64Value(result.ExpiresAt)
	data.LastUsed = types.Int64Value(result.LastUsed)
	data.CreatedAt = types.Int64Value(result.CreatedAt)
	data.UpdatedAt = types.Int64Value(result.UpdatedAt)

	elems := make([]attr.Value, len(result.Scopes))
	for i, s := range result.Scopes {
		elems[i] = types.StringValue(s)
	}
	data.Scopes, _ = types.ListValue(types.StringType, elems)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
