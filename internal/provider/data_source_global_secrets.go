package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*globalSecretsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*globalSecretsDataSource)(nil)

func NewGlobalSecretsDataSource() datasource.DataSource {
	return &globalSecretsDataSource{}
}

type globalSecretsDataSource struct {
	datasourceWithClient
}

type globalSecretsDataSourceModel struct {
	Secrets []globalSecretItemModel `tfsdk:"secrets"`
}

type globalSecretItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *globalSecretsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_secrets"
}

func (d *globalSecretsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all global secrets on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"secrets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: globalSecretSchemaAttrs(),
				},
			},
		},
	}
}

func (d *globalSecretsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	all, err := fetchAllPages[globalSecretAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/secrets", d.client.Host))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch secrets", err.Error())
		return
	}

	secrets := make([]globalSecretItemModel, len(all))
	for i, s := range all {
		secrets[i] = globalSecretItemModel{
			ID:        types.Int64Value(s.ID),
			Name:      types.StringValue(s.Name),
			Source:    types.StringValue(s.Source),
			CreatedAt: types.Int64Value(s.CreatedAt),
			UpdatedAt: types.Int64Value(s.UpdatedAt),
			Events:    stringsToList(s.Events),
			Images:    stringsToList(s.Images),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &globalSecretsDataSourceModel{Secrets: secrets})...)
}
