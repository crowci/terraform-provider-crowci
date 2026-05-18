package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*organizationSecretsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*organizationSecretsDataSource)(nil)

func NewOrganizationSecretsDataSource() datasource.DataSource {
	return &organizationSecretsDataSource{}
}

type organizationSecretsDataSource struct {
	datasourceWithClient
}

type organizationSecretsDataSourceModel struct {
	OrgID   types.Int64                   `tfsdk:"org_id"`
	Secrets []organizationSecretItemModel `tfsdk:"secrets"`
}

type organizationSecretItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *organizationSecretsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_secrets"
}

func (d *organizationSecretsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all secrets for a specific organization on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the organization to list secrets for.",
			},
			"secrets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: organizationSecretSchemaAttrs(),
				},
			},
		},
	}
}

func (d *organizationSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data organizationSecretsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	all, err := fetchAllPages[globalSecretAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/orgs/%d/secrets", d.client.Host, data.OrgID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch secrets", err.Error())
		return
	}

	secrets := make([]organizationSecretItemModel, len(all))
	for i, s := range all {
		secrets[i] = organizationSecretItemModel{
			ID:        types.Int64Value(s.ID),
			Name:      types.StringValue(s.Name),
			Source:    types.StringValue(s.Source),
			CreatedAt: types.Int64Value(s.CreatedAt),
			UpdatedAt: types.Int64Value(s.UpdatedAt),
			Events:    stringsToList(s.Events),
			Images:    stringsToList(s.Images),
		}
	}

	data.Secrets = secrets
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
