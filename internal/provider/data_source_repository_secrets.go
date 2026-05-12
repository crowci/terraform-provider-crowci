package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*repositorySecretsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositorySecretsDataSource)(nil)

func NewRepositorySecretsDataSource() datasource.DataSource {
	return &repositorySecretsDataSource{}
}

type repositorySecretsDataSource struct {
	datasourceWithClient
}

type repositorySecretsDataSourceModel struct {
	RepoID  types.Int64                  `tfsdk:"repo_id"`
	Secrets []repositorySecretItemModel  `tfsdk:"secrets"`
}

type repositorySecretItemModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (d *repositorySecretsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_secrets"
}

func (d *repositorySecretsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	secretAttrs := map[string]schema.Attribute{
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

	resp.Schema = schema.Schema{
		Description: "Fetches all secrets for a specific repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository to list secrets for.",
			},
			"secrets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: secretAttrs,
				},
			},
		},
	}
}

func (d *repositorySecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositorySecretsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	all, err := fetchAllPages[globalSecretAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/repos/%d/secrets", d.client.Host, data.RepoID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch secrets", err.Error())
		return
	}

	secrets := make([]repositorySecretItemModel, len(all))
	for i, s := range all {
		secrets[i] = repositorySecretItemModel{
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
