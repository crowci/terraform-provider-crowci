package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*repositoryRegistriesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositoryRegistriesDataSource)(nil)

func NewRepositoryRegistriesDataSource() datasource.DataSource {
	return &repositoryRegistriesDataSource{}
}

type repositoryRegistriesDataSource struct {
	datasourceWithClient
}

type repositoryRegistriesDataSourceModel struct {
	RepoID     types.Int64               `tfsdk:"repo_id"`
	Registries []repositoryRegistryItem  `tfsdk:"registries"`
}

type repositoryRegistryItem struct {
	ID        types.Int64  `tfsdk:"id"`
	Address   types.String `tfsdk:"address"`
	Username  types.String `tfsdk:"username"`
	ReadOnly  types.Bool   `tfsdk:"readonly"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *repositoryRegistriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_registries"
}

func (d *repositoryRegistriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all container registry credentials for a specific repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository to list registries for.",
			},
			"registries": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: repositoryRegistrySchemaAttrs(),
				},
			},
		},
	}
}

func (d *repositoryRegistriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryRegistriesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	all, err := fetchAllPages[registryAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/repos/%d/registries", d.client.Host, data.RepoID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch registries", err.Error())
		return
	}

	registries := make([]repositoryRegistryItem, len(all))
	for i, r := range all {
		registries[i] = repositoryRegistryItem{
			ID:        types.Int64Value(r.ID),
			Address:   types.StringValue(r.Address),
			Username:  types.StringValue(r.Username),
			ReadOnly:  types.BoolValue(r.ReadOnly),
			CreatedAt: types.Int64Value(r.CreatedAt),
			UpdatedAt: types.Int64Value(r.UpdatedAt),
		}
	}

	data.Registries = registries
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
