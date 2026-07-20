package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*userRepositoriesAvailableDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*userRepositoriesAvailableDataSource)(nil)

func NewUserRepositoriesAvailableDataSource() datasource.DataSource {
	return &userRepositoriesAvailableDataSource{}
}

type userRepositoriesAvailableDataSource struct {
	datasourceWithClient
}

type userRepositoriesAvailableDataSourceModel struct {
	ForgeID       types.Int64                    `tfsdk:"forge_id"`
	OnlyUnenabled types.Bool                     `tfsdk:"only_unenabled"`
	Repositories  []availableRepositoryItemModel `tfsdk:"repositories"`
}

type availableRepositoryItemModel struct {
	ID                           types.Int64  `tfsdk:"id"`
	ForgeRemoteID                types.String `tfsdk:"forge_remote_id"`
	ForgeID                      types.Int64  `tfsdk:"forge_id"`
	OrgID                        types.Int64  `tfsdk:"org_id"`
	Owner                        types.String `tfsdk:"owner"`
	Name                         types.String `tfsdk:"name"`
	FullName                     types.String `tfsdk:"full_name"`
	CloneURL                     types.String `tfsdk:"clone_url"`
	CloneURLSSH                  types.String `tfsdk:"clone_url_ssh"`
	ForgeURL                     types.String `tfsdk:"forge_url"`
	AvatarURL                    types.String `tfsdk:"avatar_url"`
	DefaultBranch                types.String `tfsdk:"default_branch"`
	Active                       types.Bool   `tfsdk:"active"`
	Private                      types.Bool   `tfsdk:"private"`
	PREnabled                    types.Bool   `tfsdk:"pr_enabled"`
	AllowDeploy                  types.Bool   `tfsdk:"allow_deploy"`
	AllowManual                  types.Bool   `tfsdk:"allow_manual"`
	AllowPR                      types.Bool   `tfsdk:"allow_pr"`
	Visibility                   types.String `tfsdk:"visibility"`
	RequireApproval              types.String `tfsdk:"require_approval"`
	Trusted                      types.Object `tfsdk:"trusted"`
	ConfigFile                   types.String `tfsdk:"config_file"`
	DeployTeam                   types.String `tfsdk:"deploy_team"`
	Timeout                      types.Int64  `tfsdk:"timeout"`
	CancelPreviousPipelineEvents types.List   `tfsdk:"cancel_previous_pipeline_events"`
	NetrcTrusted                 types.List   `tfsdk:"netrc_trusted"`
	LogsKeepDuration             types.String `tfsdk:"logs_keep_duration"`
	LogsKeepMin                  types.Int64  `tfsdk:"logs_keep_min"`
}

func (d *userRepositoriesAvailableDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_repositories_available"
}

func (d *userRepositoriesAvailableDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches repositories visible from linked forges (both enabled and not-yet-enabled).",
		Attributes: map[string]schema.Attribute{
			"forge_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Scope results to a specific forge ID. Defaults to all linked forges.",
			},
			"only_unenabled": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, omit repositories already enabled in Crow CI.",
			},
			"repositories": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: repositorySchemaAttrs(),
				},
			},
		},
	}
}

func (d *userRepositoriesAvailableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data userRepositoriesAvailableDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := url.Values{}
	if !data.ForgeID.IsNull() && !data.ForgeID.IsUnknown() {
		params.Set("forge_id", fmt.Sprintf("%d", data.ForgeID.ValueInt64()))
	}
	if !data.OnlyUnenabled.IsNull() && !data.OnlyUnenabled.IsUnknown() {
		params.Set("only_unenabled", fmt.Sprintf("%t", data.OnlyUnenabled.ValueBool()))
	}

	endpoint := fmt.Sprintf("%s/api/v1/user/repos/available", d.client.Host)
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	httpResp, ok := doRequest(ctx, d.client, http.MethodGet, endpoint, nil, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok {
		return
	}
	defer httpResp.Body.Close()

	var results []repositoryAPIResponse
	if !decodeJSON(httpResp.Body, &results, &resp.Diagnostics) {
		return
	}

	repos := make([]availableRepositoryItemModel, len(results))
	for i, r := range results {
		trusted, _ := types.ObjectValue(trustedAttrTypes, map[string]attr.Value{
			"network":  types.BoolValue(r.Trusted.Network),
			"security": types.BoolValue(r.Trusted.Security),
			"volumes":  types.BoolValue(r.Trusted.Volumes),
		})
		repos[i] = availableRepositoryItemModel{
			ID:                           types.Int64Value(r.ID),
			ForgeRemoteID:                types.StringValue(r.ForgeRemoteID),
			ForgeID:                      types.Int64Value(r.ForgeID),
			OrgID:                        types.Int64Value(r.OrgID),
			Owner:                        types.StringValue(r.Owner),
			Name:                         types.StringValue(r.Name),
			FullName:                     types.StringValue(r.FullName),
			CloneURL:                     types.StringValue(r.CloneURL),
			CloneURLSSH:                  types.StringValue(r.CloneURLSSH),
			ForgeURL:                     types.StringValue(r.ForgeURL),
			AvatarURL:                    types.StringValue(r.AvatarURL),
			DefaultBranch:                types.StringValue(r.DefaultBranch),
			Active:                       types.BoolValue(r.Active),
			Private:                      types.BoolValue(r.Private),
			PREnabled:                    types.BoolValue(r.PREnabled),
			AllowDeploy:                  types.BoolValue(r.AllowDeploy),
			AllowManual:                  types.BoolValue(r.AllowManual),
			AllowPR:                      types.BoolValue(r.AllowPR),
			Visibility:                   types.StringValue(r.Visibility),
			RequireApproval:              types.StringValue(r.RequireApproval),
			Trusted:                      trusted,
			ConfigFile:                   types.StringValue(r.ConfigFile),
			DeployTeam:                   types.StringValue(r.DeployTeam),
			Timeout:                      types.Int64Value(r.Timeout),
			CancelPreviousPipelineEvents: stringsToList(r.CancelPreviousPipelineEvents),
			NetrcTrusted:                 stringsToList(r.NetrcTrusted),
			LogsKeepDuration:             types.StringValue(r.LogsKeepDuration),
			LogsKeepMin:                  types.Int64Value(r.LogsKeepMin),
		}
	}

	data.Repositories = repos
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
