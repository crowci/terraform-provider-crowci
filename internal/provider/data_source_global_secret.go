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

var _ datasource.DataSource = (*globalSecretDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*globalSecretDataSource)(nil)

func NewGlobalSecretDataSource() datasource.DataSource {
	return &globalSecretDataSource{}
}

type globalSecretDataSource struct {
	client *crowciClient
}

type globalSecretDataSourceModel struct {
	Name      types.String `tfsdk:"name"`
	ID        types.Int64  `tfsdk:"id"`
	Events    types.List   `tfsdk:"events"`
	Images    types.List   `tfsdk:"images"`
	OrgID     types.Int64  `tfsdk:"org_id"`
	RepoID    types.Int64  `tfsdk:"repo_id"`
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

func (d *globalSecretDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_secret"
}

func (d *globalSecretDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a global secret by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The secret's name.",
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Secret ID.",
			},
			"events": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Events that trigger the secret.",
			},
			"images": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Images the secret is available to.",
			},
			"org_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Org scope of the secret.",
			},
			"repo_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Repo scope of the secret.",
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
		},
	}
}

func (d *globalSecretDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *globalSecretDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data globalSecretDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/secrets/%s", d.client.Host, data.Name.ValueString())
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

	if httpResp.StatusCode == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Secret not found",
			fmt.Sprintf("No global secret with name %q exists.", data.Name.ValueString()),
		)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("GET /secrets/%s returned status %d", data.Name.ValueString(), httpResp.StatusCode),
		)
		return
	}

	var result globalSecretAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	data.ID = types.Int64Value(result.ID)
	data.OrgID = int64NullIfZero(result.OrgID)
	data.RepoID = int64NullIfZero(result.RepoID)
	data.Source = types.StringValue(result.Source)
	data.CreatedAt = types.Int64Value(result.CreatedAt)
	data.UpdatedAt = types.Int64Value(result.UpdatedAt)
	data.Events = stringsToList(result.Events)
	data.Images = stringsToList(result.Images)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func stringsToList(ss []string) types.List {
	if ss == nil {
		ss = []string{}
	}
	elems := make([]types.String, len(ss))
	for i, s := range ss {
		elems[i] = types.StringValue(s)
	}
	listVal, _ := types.ListValueFrom(context.Background(), types.StringType, elems)
	return listVal
}
