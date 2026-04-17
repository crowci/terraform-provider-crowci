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

var _ datasource.DataSource = (*forgesDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*forgesDataSource)(nil)

func NewForgesDataSource() datasource.DataSource {
	return &forgesDataSource{}
}

type forgesDataSource struct {
	client *crowciClient
}

type forgesDataSourceModel struct {
	Forges []forgeModel `tfsdk:"forges"`
}

type forgeModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Type       types.String `tfsdk:"type"`
	URL        types.String `tfsdk:"url"`
	Client     types.String `tfsdk:"client"`
	Icon       types.String `tfsdk:"icon"`
	OAuthHost  types.String `tfsdk:"oauth_host"`
	SkipVerify types.Bool   `tfsdk:"skip_verify"`
}

type forgeAPIResponse struct {
	ID         int64  `json:"id"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	Client     string `json:"client"`
	Icon       string `json:"icon"`
	OAuthHost  string `json:"oauth_host"`
	SkipVerify bool   `json:"skip_verify"`
}

func (d *forgesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_forges"
}

func (d *forgesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of forges configured in Crow CI.",
		Attributes: map[string]schema.Attribute{
			"forges": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed:    true,
							Description: "Forge ID.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "Forge type (e.g. github, gitlab, gitea).",
						},
						"url": schema.StringAttribute{
							Computed:    true,
							Description: "Base URL of the forge.",
						},
						"client": schema.StringAttribute{
							Computed:    true,
							Description: "OAuth client ID.",
						},
						"icon": schema.StringAttribute{
							Computed:    true,
							Description: "Icon URL for the forge.",
						},
						"oauth_host": schema.StringAttribute{
							Computed:    true,
							Description: "OAuth host if different from the forge URL.",
						},
						"skip_verify": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether TLS verification is skipped.",
						},
					},
				},
			},
		},
	}
}

func (d *forgesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *forgesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	var all []forgeAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/forges?page=%d&perPage=50", d.client.Host, page)
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
				fmt.Sprintf("GET /forges returned status %d", httpResp.StatusCode),
			)
			return
		}

		var page_results []forgeAPIResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&page_results); err != nil {
			resp.Diagnostics.AddError("Failed to decode response", err.Error())
			return
		}

		all = append(all, page_results...)
		if len(page_results) < 50 {
			break
		}
	}

	forges := make([]forgeModel, len(all))
	for i, f := range all {
		forges[i] = forgeModel{
			ID:         types.Int64Value(f.ID),
			Type:       types.StringValue(f.Type),
			URL:        types.StringValue(f.URL),
			Client:     types.StringValue(f.Client),
			Icon:       types.StringValue(f.Icon),
			OAuthHost:  types.StringValue(f.OAuthHost),
			SkipVerify: types.BoolValue(f.SkipVerify),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &forgesDataSourceModel{Forges: forges})...)
}
