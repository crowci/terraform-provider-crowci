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

var _ datasource.DataSource = (*organizationDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*organizationDataSource)(nil)

func NewOrganizationDataSource() datasource.DataSource {
	return &organizationDataSource{}
}

type organizationDataSource struct {
	client *crowciClient
}

type organizationDataSourceModel struct {
	ID      types.Int64  `tfsdk:"id"`
	ForgeID types.Int64  `tfsdk:"forge_id"`
	Name    types.String `tfsdk:"name"`
	IsUser  types.Bool   `tfsdk:"is_user"`
}

type organizationAPIResponse struct {
	ID      int64  `json:"id"`
	ForgeID int64  `json:"forge_id"`
	Name    string `json:"name"`
	IsUser  bool   `json:"is_user"`
}

func (d *organizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *organizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get an organization by ID. The resource will still return an organization even if it does not exists, but it will return with id = 0",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "The organization's ID.",
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
		},
	}
}

func (d *organizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%d", d.client.Host, data.ID.ValueInt64())
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

	if httpResp.StatusCode == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Organization not found",
			fmt.Sprintf("No organization with ID %d exists.", data.ID.ValueInt64()),
		)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("GET /orgs/%d returned status %d", data.ID.ValueInt64(), httpResp.StatusCode),
		)
		return
	}

	var result organizationAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	data.ForgeID = int64NullIfZero(result.ForgeID)
	data.Name = types.StringValue(result.Name)
	data.IsUser = types.BoolValue(result.IsUser)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
