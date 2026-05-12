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

var _ datasource.DataSource = (*repositoryRegistriesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositoryRegistriesDataSource)(nil)

func NewRepositoryRegistriesDataSource() datasource.DataSource {
	return &repositoryRegistriesDataSource{}
}

type repositoryRegistriesDataSource struct {
	client *crowciClient
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
		Description: "Fetches all container registry credentials for a specific repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository to list registries for.",
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

func (d *repositoryRegistriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *repositoryRegistriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryRegistriesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var all []registryAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/repos/%d/registries?page=%d&perPage=50", d.client.Host, data.RepoID.ValueInt64(), page)
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
				fmt.Sprintf("GET /repos/%d/registries returned status %d", data.RepoID.ValueInt64(), httpResp.StatusCode),
			)
			return
		}

		var pageResults []registryAPIResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&pageResults); err != nil {
			resp.Diagnostics.AddError("Failed to decode response", err.Error())
			return
		}

		all = append(all, pageResults...)
		if len(pageResults) < 50 {
			break
		}
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
