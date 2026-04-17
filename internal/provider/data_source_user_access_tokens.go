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

var _ datasource.DataSource = (*userAccessTokensDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*userAccessTokensDataSource)(nil)

func NewUserAccessTokensDataSource() datasource.DataSource {
	return &userAccessTokensDataSource{}
}

type userAccessTokensDataSource struct {
	client *crowciClient
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
	tokenAttrs := map[string]schema.Attribute{
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
	}

	resp.Schema = schema.Schema{
		Description: "Fetches all access tokens for the currently authenticated user.",
		Attributes: map[string]schema.Attribute{
			"tokens": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: tokenAttrs,
				},
			},
		},
	}
}

func (d *userAccessTokensDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userAccessTokensDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var all []accessTokenAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/user/access-tokens?page=%d&perPage=50", d.client.Host, page)
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
				fmt.Sprintf("GET /user/access-tokens returned status %d", httpResp.StatusCode),
			)
			return
		}

		var pageResults []accessTokenAPIResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&pageResults); err != nil {
			resp.Diagnostics.AddError("Failed to decode response", err.Error())
			return
		}

		all = append(all, pageResults...)
		if len(pageResults) < 50 {
			break
		}
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
