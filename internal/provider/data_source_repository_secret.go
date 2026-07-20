package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*repositorySecretDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositorySecretDataSource)(nil)

func NewRepositorySecretDataSource() datasource.DataSource {
	return &repositorySecretDataSource{}
}

type repositorySecretDataSource struct {
	datasourceWithClient
}

type repositorySecretDataSourceModel struct {
	RepoID    types.Int64  `tfsdk:"repo_id"`
	Name      types.String `tfsdk:"name"`
	ID        types.Int64  `tfsdk:"id"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func repositorySecretSchemaAttrs() map[string]schema.Attribute {
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

func (d *repositorySecretDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_secret"
}

func (d *repositorySecretDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := repositorySecretSchemaAttrs()
	attrs["repo_id"] = schema.Int64Attribute{
		Required:    true,
		Description: "ID of the repository the secret belongs to.",
	}
	attrs["name"] = schema.StringAttribute{
		Required:    true,
		Description: "The secret's name.",
	}
	resp.Schema = schema.Schema{
		Description: "Get a secret scoped to a specific repository by name.",
		Attributes:  attrs,
	}
}

func (d *repositorySecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositorySecretDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/secrets/%s", d.client.Host, data.RepoID.ValueInt64(), data.Name.ValueString())
	httpResp, ok := doRequest(ctx, d.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Secret not found",
			fmt.Sprintf("No secret with name %q exists in repository %d.", data.Name.ValueString(), data.RepoID.ValueInt64()),
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
