package provider

import (
	"context"
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
	datasourceWithClient
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

func organizationRegistrySchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Registry ID.",
		},
		"address": schema.StringAttribute{
			Computed:    true,
			Description: "Registry address.",
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
	}
}

func (d *organizationRegistryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_registry"
}

func (d *organizationRegistryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := organizationRegistrySchemaAttrs()
	attrs["org_id"] = schema.Int64Attribute{
		Required:    true,
		Description: "ID of the organization this registry belongs to.",
	}
	attrs["address"] = schema.StringAttribute{
		Required:    true,
		Description: "Registry address (e.g. 'docker.io', 'ghcr.io').",
	}
	attrs["id"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Registry ID assigned by Crow CI.",
	}
	resp.Schema = schema.Schema{
		Description: "Get a container registry credential for an organization by org ID and registry address.",
		Attributes:  attrs,
	}
}

func (d *organizationRegistryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationRegistryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%d/registries/%s", d.client.Host, data.OrgID.ValueInt64(), data.Address.ValueString())
	httpResp, ok := doRequest(ctx, d.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Registry not found",
			fmt.Sprintf("No registry %q exists for organization %d.", data.Address.ValueString(), data.OrgID.ValueInt64()),
		)
		return
	}

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	data.ID = types.Int64Value(result.ID)
	data.Username = types.StringValue(result.Username)
	data.ReadOnly = types.BoolValue(result.ReadOnly)
	data.CreatedAt = types.Int64Value(result.CreatedAt)
	data.UpdatedAt = types.Int64Value(result.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
