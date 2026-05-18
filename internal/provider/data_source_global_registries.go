package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*globalRegistriesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*globalRegistriesDataSource)(nil)

func NewGlobalRegistriesDataSource() datasource.DataSource {
	return &globalRegistriesDataSource{}
}

type globalRegistriesDataSource struct {
	datasourceWithClient
}

type globalRegistriesDataSourceModel struct {
	Registries []globalRegistryItemModel `tfsdk:"registries"`
}

type globalRegistryItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Address   types.String `tfsdk:"address"`
	Username  types.String `tfsdk:"username"`
	ReadOnly  types.Bool   `tfsdk:"readonly"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *globalRegistriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_registries"
}

func (d *globalRegistriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all global container registry credentials on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"registries": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: globalRegistrySchemaAttrs(),
				},
			},
		},
	}
}

func (d *globalRegistriesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	all, err := fetchAllPages[registryAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/registries", d.client.Host))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch registries", err.Error())
		return
	}

	registries := make([]globalRegistryItemModel, len(all))
	for i, r := range all {
		registries[i] = globalRegistryItemModel{
			ID:        types.Int64Value(r.ID),
			Address:   types.StringValue(r.Address),
			Username:  types.StringValue(r.Username),
			ReadOnly:  types.BoolValue(r.ReadOnly),
			CreatedAt: types.Int64Value(r.CreatedAt),
			UpdatedAt: types.Int64Value(r.UpdatedAt),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &globalRegistriesDataSourceModel{Registries: registries})...)
}
