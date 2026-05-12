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

var _ datasource.DataSource = (*globalRegistriesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*globalRegistriesDataSource)(nil)

func NewGlobalRegistriesDataSource() datasource.DataSource {
	return &globalRegistriesDataSource{}
}

type globalRegistriesDataSource struct {
	client *crowciClient
}

type globalRegistriesDataSourceModel struct {
	Registries []globalRegistryItemModel `tfsdk:"registries"`
}

type globalRegistryItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	OrgID     types.Int64  `tfsdk:"org_id"`
	RepoID    types.Int64  `tfsdk:"repo_id"`
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
	registryAttrs := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Registry ID.",
		},
		"org_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Organization scope of the registry.",
		},
		"repo_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Repo scope of the registry.",
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
		Description: "Fetches all global container registry credentials on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"registries": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: registryAttrs,
				},
			},
		},
	}
}

func (d *globalRegistriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *globalRegistriesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var all []registryAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/registries?page=%d&perPage=50", d.client.Host, page)
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
				fmt.Sprintf("GET /registries returned status %d", httpResp.StatusCode),
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

	registries := make([]globalRegistryItemModel, len(all))
	for i, r := range all {
		registries[i] = globalRegistryItemModel{
			ID:        types.Int64Value(r.ID),
			OrgID:     int64NullIfZero(r.OrgID),
			RepoID:    int64NullIfZero(r.RepoID),
			Address:   types.StringValue(r.Address),
			Username:  types.StringValue(r.Username),
			ReadOnly:  types.BoolValue(r.ReadOnly),
			CreatedAt: types.Int64Value(r.CreatedAt),
			UpdatedAt: types.Int64Value(r.UpdatedAt),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &globalRegistriesDataSourceModel{Registries: registries})...)
}
