package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*organizationRegistriesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*organizationRegistriesDataSource)(nil)

func NewOrganizationRegistriesDataSource() datasource.DataSource {
	return &organizationRegistriesDataSource{}
}

type organizationRegistriesDataSource struct {
	datasourceWithClient
}

type organizationRegistriesDataSourceModel struct {
	OrgID      types.Int64             `tfsdk:"org_id"`
	Registries []registryItemModel     `tfsdk:"registries"`
}

type registryItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Address   types.String `tfsdk:"address"`
	Username  types.String `tfsdk:"username"`
	ReadOnly  types.Bool   `tfsdk:"readonly"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *organizationRegistriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_registries"
}

func (d *organizationRegistriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	registryAttrs := map[string]schema.Attribute{
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

	resp.Schema = schema.Schema{
		Description: "Fetches all container registry credentials for a specific organization on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the organization to list registries for.",
			},
			"registries": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: registryAttrs,
				},
			},
		},
	}
}

func (d *organizationRegistriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationRegistriesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	all, err := fetchAllPages[registryAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/orgs/%d/registries", d.client.Host, data.OrgID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch registries", err.Error())
		return
	}

	registries := make([]registryItemModel, len(all))
	for i, r := range all {
		registries[i] = registryItemModel{
			ID:      types.Int64Value(r.ID),
			Address: types.StringValue(r.Address),
			Username:  types.StringValue(r.Username),
			ReadOnly:  types.BoolValue(r.ReadOnly),
			CreatedAt: types.Int64Value(r.CreatedAt),
			UpdatedAt: types.Int64Value(r.UpdatedAt),
		}
	}

	data.Registries = registries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
