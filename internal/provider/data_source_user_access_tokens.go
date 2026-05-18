package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*userAccessTokensDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*userAccessTokensDataSource)(nil)

func NewUserAccessTokensDataSource() datasource.DataSource {
	return &userAccessTokensDataSource{}
}

type userAccessTokensDataSource struct {
	datasourceWithClient
}

type userAccessTokensDataSourceModel struct {
	Tokens []userAccessTokenItemModel `tfsdk:"tokens"`
}

type userAccessTokenItemModel struct {
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

func (d *userAccessTokensDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_access_tokens"
}

func (d *userAccessTokensDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all access tokens for the currently authenticated user.",
		Attributes: map[string]schema.Attribute{
			"tokens": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: userAccessTokenSchemaAttrs(),
				},
			},
		},
	}
}

func (d *userAccessTokensDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	all, err := fetchAllPages[accessTokenAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/user/access-tokens", d.client.Host))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch access tokens", err.Error())
		return
	}

	tokens := make([]userAccessTokenItemModel, len(all))
	for i, t := range all {
		elems := make([]attr.Value, len(t.Scopes))
		for j, s := range t.Scopes {
			elems[j] = types.StringValue(s)
		}
		scopes, _ := types.ListValue(types.StringType, elems)

		tokens[i] = userAccessTokenItemModel{
			ID:        types.Int64Value(t.ID),
			Name:      types.StringValue(t.Name),
			Scopes:    scopes,
			UserID:    types.Int64Value(t.UserID),
			OrgID:     types.Int64Value(t.OrgID),
			RepoID:    types.Int64Value(t.RepoID),
			ExpiresAt: types.Int64Value(t.ExpiresAt),
			LastUsed:  types.Int64Value(t.LastUsed),
			CreatedAt: types.Int64Value(t.CreatedAt),
			UpdatedAt: types.Int64Value(t.UpdatedAt),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &userAccessTokensDataSourceModel{Tokens: tokens})...)
}
