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

var _ datasource.DataSource = (*organizationRegistryDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*organizationRegistryDataSource)(nil)

func NewOrganizationRegistryDataSource() datasource.DataSource {
	return &organizationRegistryDataSource{}
}

type organizationRegistryDataSource struct {
	client *crowciClient
}

type organizationRegistryDataSourceModel struct {
	OrgID     types.Int64  `tfsdk:"org_id"`
	Address   types.String `tfsdk:"address"`
	ID        types.Int64  `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	ReadOnly  types.Bool   `tfsdk:"readonly"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *organizationRegistryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_registry"
}

func (d *organizationRegistryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a container registry credential for an organization by org ID and registry address.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the organization this registry belongs to.",
			},
			"address": schema.StringAttribute{
				Required:    true,
				Description: "Registry address (e.g. 'docker.io', 'ghcr.io').",
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Registry ID assigned by Crow CI.",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "Registry username.",
			},
			"readonly": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the registry is read-only.",
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

func (d *organizationRegistryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *organizationRegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationRegistryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%d/registries/%s", d.client.Host, data.OrgID.ValueInt64(), data.Address.ValueString())
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
			"Registry not found",
			fmt.Sprintf("No registry %q exists for organization %d.", data.Address.ValueString(), data.OrgID.ValueInt64()),
		)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("GET /orgs/%d/registries/%s returned status %d", data.OrgID.ValueInt64(), data.Address.ValueString(), httpResp.StatusCode),
		)
		return
	}

	var result registryAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	data.ID = types.Int64Value(result.ID)
	data.Username = types.StringValue(result.Username)
	data.ReadOnly = types.BoolValue(result.ReadOnly)
	data.CreatedAt = types.Int64Value(result.CreatedAt)
	data.UpdatedAt = types.Int64Value(result.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
