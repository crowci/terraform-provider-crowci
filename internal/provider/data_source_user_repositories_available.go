package provider

import (
	"context"
	"encoding/json"
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
	client *crowciClient
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
	trustedAttrs := map[string]schema.Attribute{
		"network": schema.BoolAttribute{
			Computed:    true,
			Description: "Allow network access for trusted operations.",
		},
		"security": schema.BoolAttribute{
			Computed:    true,
			Description: "Allow security-sensitive operations.",
		},
		"volumes": schema.BoolAttribute{
			Computed:    true,
			Description: "Allow volume mounts.",
		},
	}

	repoAttrs := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Repository ID.",
		},
		"forge_remote_id": schema.StringAttribute{
			Computed:    true,
			Description: "Repository ID at the forge.",
		},
		"forge_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Forge ID.",
		},
		"org_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Organization ID the repository belongs to.",
		},
		"owner": schema.StringAttribute{
			Computed:    true,
			Description: "Repository owner.",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Repository name.",
		},
		"full_name": schema.StringAttribute{
			Computed:    true,
			Description: "Full repository name (owner/name).",
		},
		"clone_url": schema.StringAttribute{
			Computed:    true,
			Description: "HTTPS clone URL.",
		},
		"clone_url_ssh": schema.StringAttribute{
			Computed:    true,
			Description: "SSH clone URL.",
		},
		"forge_url": schema.StringAttribute{
			Computed:    true,
			Description: "URL of the repository at the forge.",
		},
		"avatar_url": schema.StringAttribute{
			Computed:    true,
			Description: "Avatar URL for the repository.",
		},
		"default_branch": schema.StringAttribute{
			Computed:    true,
			Description: "Default branch of the repository.",
		},
		"active": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the repository is already active in Crow CI.",
		},
		"private": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the repository is private.",
		},
		"pr_enabled": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether pull request pipelines are enabled.",
		},
		"allow_deploy": schema.BoolAttribute{
			Computed:    true,
			Description: "Allow deploy pipelines.",
		},
		"allow_manual": schema.BoolAttribute{
			Computed:    true,
			Description: "Allow manual pipeline triggers.",
		},
		"allow_pr": schema.BoolAttribute{
			Computed:    true,
			Description: "Allow pull request pipelines.",
		},
		"visibility": schema.StringAttribute{
			Computed:    true,
			Description: "Repository visibility.",
		},
		"require_approval": schema.StringAttribute{
			Computed:    true,
			Description: "Approval requirement for pipelines.",
		},
		"trusted": schema.SingleNestedAttribute{
			Computed:    true,
			Description: "Trusted configuration for the repository.",
			Attributes:  trustedAttrs,
		},
		"config_file": schema.StringAttribute{
			Computed:    true,
			Description: "Path to the pipeline configuration file.",
		},
		"deploy_team": schema.StringAttribute{
			Computed:    true,
			Description: "Team allowed to trigger deploy pipelines.",
		},
		"timeout": schema.Int64Attribute{
			Computed:    true,
			Description: "Pipeline execution timeout in minutes.",
		},
		"cancel_previous_pipeline_events": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Pipeline events for which previous runs should be cancelled.",
		},
		"netrc_trusted": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Hostnames for which .netrc credentials are trusted.",
		},
		"logs_keep_duration": schema.StringAttribute{
			Computed:    true,
			Description: "Duration to keep pipeline logs.",
		},
		"logs_keep_min": schema.Int64Attribute{
			Computed:    true,
			Description: "Minimum number of pipeline logs to keep.",
		},
	}

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
					Attributes: repoAttrs,
				},
			},
		},
	}
}

func (d *userRepositoriesAvailableDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
			fmt.Sprintf("GET /user/repos/available returned status %d", httpResp.StatusCode),
		)
		return
	}

	var results []repositoryAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&results); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
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
