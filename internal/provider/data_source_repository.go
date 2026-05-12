package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*repositoryDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositoryDataSource)(nil)

func NewRepositoryDataSource() datasource.DataSource {
	return &repositoryDataSource{}
}

type repositoryDataSource struct {
	datasourceWithClient
}

type repositoryDataSourceModel struct {
	ID                           types.Int64         `tfsdk:"id"`
	ForgeRemoteID                types.String        `tfsdk:"forge_remote_id"`
	ForgeID                      types.Int64         `tfsdk:"forge_id"`
	OrgID                        types.Int64         `tfsdk:"org_id"`
	Owner                        types.String        `tfsdk:"owner"`
	Name                         types.String        `tfsdk:"name"`
	FullName                     types.String        `tfsdk:"full_name"`
	CloneURL                     types.String        `tfsdk:"clone_url"`
	CloneURLSSH                  types.String        `tfsdk:"clone_url_ssh"`
	ForgeURL                     types.String        `tfsdk:"forge_url"`
	AvatarURL                    types.String        `tfsdk:"avatar_url"`
	DefaultBranch                types.String        `tfsdk:"default_branch"`
	Active                       types.Bool          `tfsdk:"active"`
	Private                      types.Bool          `tfsdk:"private"`
	PREnabled                    types.Bool          `tfsdk:"pr_enabled"`
	AllowDeploy                  types.Bool          `tfsdk:"allow_deploy"`
	AllowManual                  types.Bool          `tfsdk:"allow_manual"`
	AllowPR                      types.Bool          `tfsdk:"allow_pr"`
	Visibility                   types.String        `tfsdk:"visibility"`
	RequireApproval              types.String        `tfsdk:"require_approval"`
	Trusted                      types.Object        `tfsdk:"trusted"`
	ConfigFile                   types.String        `tfsdk:"config_file"`
	DeployTeam                   types.String        `tfsdk:"deploy_team"`
	Timeout                      types.Int64         `tfsdk:"timeout"`
	CancelPreviousPipelineEvents types.List          `tfsdk:"cancel_previous_pipeline_events"`
	NetrcTrusted                 types.List          `tfsdk:"netrc_trusted"`
	LogsKeepDuration             types.String        `tfsdk:"logs_keep_duration"`
	LogsKeepMin                  types.Int64         `tfsdk:"logs_keep_min"`
}

func (d *repositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *repositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a repository by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "Repository ID.",
			},
			"forge_remote_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the repository at the forge.",
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
				Description: "Whether the repository is active.",
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
				Attributes: map[string]schema.Attribute{
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
				},
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
		},
	}
}

func (d *repositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d", d.client.Host, data.ID.ValueInt64())
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
			"Repository not found",
			fmt.Sprintf("No repository with ID %d exists.", data.ID.ValueInt64()),
		)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("GET /repos/%d returned status %d", data.ID.ValueInt64(), httpResp.StatusCode),
		)
		return
	}

	var result repositoryAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	data.ForgeRemoteID = types.StringValue(result.ForgeRemoteID)
	data.ForgeID = types.Int64Value(result.ForgeID)
	data.OrgID = types.Int64Value(result.OrgID)
	data.Owner = types.StringValue(result.Owner)
	data.Name = types.StringValue(result.Name)
	data.FullName = types.StringValue(result.FullName)
	data.CloneURL = types.StringValue(result.CloneURL)
	data.CloneURLSSH = types.StringValue(result.CloneURLSSH)
	data.ForgeURL = types.StringValue(result.ForgeURL)
	data.AvatarURL = types.StringValue(result.AvatarURL)
	data.DefaultBranch = types.StringValue(result.DefaultBranch)
	data.Active = types.BoolValue(result.Active)
	data.Private = types.BoolValue(result.Private)
	data.PREnabled = types.BoolValue(result.PREnabled)
	data.AllowDeploy = types.BoolValue(result.AllowDeploy)
	data.AllowManual = types.BoolValue(result.AllowManual)
	data.AllowPR = types.BoolValue(result.AllowPR)
	data.Visibility = types.StringValue(result.Visibility)
	data.RequireApproval = types.StringValue(result.RequireApproval)
	data.ConfigFile = types.StringValue(result.ConfigFile)
	data.DeployTeam = types.StringValue(result.DeployTeam)
	data.Timeout = types.Int64Value(result.Timeout)
	data.CancelPreviousPipelineEvents = stringsToList(result.CancelPreviousPipelineEvents)
	data.NetrcTrusted = stringsToList(result.NetrcTrusted)
	data.LogsKeepDuration = types.StringValue(result.LogsKeepDuration)
	data.LogsKeepMin = types.Int64Value(result.LogsKeepMin)
	data.Trusted, _ = types.ObjectValue(trustedAttrTypes, map[string]attr.Value{
		"network":  types.BoolValue(result.Trusted.Network),
		"security": types.BoolValue(result.Trusted.Security),
		"volumes":  types.BoolValue(result.Trusted.Volumes),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
