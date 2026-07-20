package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*globalSecretDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*globalSecretDataSource)(nil)

func NewGlobalSecretDataSource() datasource.DataSource {
	return &globalSecretDataSource{}
}

type globalSecretDataSource struct {
	datasourceWithClient
}

type globalSecretDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	ID        types.Int64  `tfsdk:"id"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

type globalSecretAPIResponse struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Events    []string `json:"events"`
	Images    []string `json:"images"`
	OrgID     int64    `json:"org_id"`
	RepoID    int64    `json:"repo_id"`
	Source    string   `json:"source"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

func globalSecretSchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Secret ID.",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Secret name.",
		},
		"events": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Events that trigger the secret.",
		},
		"images": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Container images the secret is available to.",
		},
		"source": schema.StringAttribute{
			Computed:    true,
			Description: "Source of the secret.",
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

func (d *globalSecretDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_secret"
}

func (d *globalSecretDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := globalSecretSchemaAttrs()
	attrs["name"] = schema.StringAttribute{
		Required:    true,
		Description: "The secret's name.",
	}
	resp.Schema = schema.Schema{
		Description: "Get a global secret by name.",
		Attributes:  attrs,
	}
}

func (d *globalSecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data globalSecretDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/secrets/%s", d.client.Host, data.Name.ValueString())
	httpResp, ok := doRequest(ctx, d.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Secret not found",
			fmt.Sprintf("No global secret with name %q exists.", data.Name.ValueString()),
		)
		return
	}

	var result globalSecretAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) {
		return
	}

	data.ID = types.Int64Value(result.ID)
	data.Source = types.StringValue(result.Source)
	data.CreatedAt = types.Int64Value(result.CreatedAt)
	data.UpdatedAt = types.Int64Value(result.UpdatedAt)
	data.Events = stringsToList(result.Events)
	data.Images = stringsToList(result.Images)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
