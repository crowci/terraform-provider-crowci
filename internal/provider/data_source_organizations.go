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

var _ datasource.DataSource = (*organizationsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*organizationsDataSource)(nil)

func NewOrganizationsDataSource() datasource.DataSource {
	return &organizationsDataSource{}
}

type organizationsDataSource struct {
	client *crowciClient
}

type organizationsDataSourceModel struct {
	Organizations []organizationItemModel `tfsdk:"organizations"`
}

type organizationItemModel struct {
	ID      types.Int64  `tfsdk:"id"`
	ForgeID types.Int64  `tfsdk:"forge_id"`
	Name    types.String `tfsdk:"name"`
	IsUser  types.Bool   `tfsdk:"is_user"`
}

func (d *organizationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organizations"
}

func (d *organizationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	orgAttrs := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Organization ID.",
		},
		"forge_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Forge ID the organization belongs to.",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Organization name.",
		},
		"is_user": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether this organization represents a user account.",
		},
	}

	resp.Schema = schema.Schema{
		Description: "Fetches all organizations accessible to the authenticated user.",
		Attributes: map[string]schema.Attribute{
			"organizations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: orgAttrs,
				},
			},
		},
	}
}

func (d *organizationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var all []organizationAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/orgs?page=%d&perPage=50", d.client.Host, page)
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
				fmt.Sprintf("GET /orgs returned status %d", httpResp.StatusCode),
			)
			return
		}

		var pageResults []organizationAPIResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&pageResults); err != nil {
			resp.Diagnostics.AddError("Failed to decode response", err.Error())
			return
		}

		all = append(all, pageResults...)
		if len(pageResults) < 50 {
			break
		}
	}

	orgs := make([]organizationItemModel, len(all))
	for i, o := range all {
		orgs[i] = organizationItemModel{
			ID:      types.Int64Value(o.ID),
			ForgeID: int64NullIfZero(o.ForgeID),
			Name:    types.StringValue(o.Name),
			IsUser:  types.BoolValue(o.IsUser),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &organizationsDataSourceModel{Organizations: orgs})...)
}
