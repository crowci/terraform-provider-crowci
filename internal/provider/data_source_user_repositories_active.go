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

var lastPipelineAttrTypes = map[string]attr.Type{
	"id":           types.Int64Type,
	"number":       types.Int64Type,
	"status":       types.StringType,
	"event":        types.StringType,
	"branch":       types.StringType,
	"commit":       types.StringType,
	"ref":          types.StringType,
	"message":      types.StringType,
	"author":       types.StringType,
	"author_email": types.StringType,
	"sender":       types.StringType,
	"forge_url":    types.StringType,
	"created":      types.Int64Type,
	"started":      types.Int64Type,
	"finished":     types.Int64Type,
}

var _ datasource.DataSource = (*userRepositoriesActiveDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*userRepositoriesActiveDataSource)(nil)

func NewUserRepositoriesActiveDataSource() datasource.DataSource {
	return &userRepositoriesActiveDataSource{}
}

type userRepositoriesActiveDataSource struct {
	client *crowciClient
}

type userRepositoriesActiveDataSourceModel struct {
	Repositories []activeRepositoryItemModel `tfsdk:"repositories"`
}

type activeRepositoryItemModel struct {
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
	LastPipeline                 types.Object `tfsdk:"last_pipeline"`
}

type repoLastPipelineAPIResponse struct {
	ID                           int64                `json:"id"`
	ForgeRemoteID                string               `json:"forge_remote_id"`
	ForgeID                      int64                `json:"forge_id"`
	OrgID                        int64                `json:"org_id"`
	Owner                        string               `json:"owner"`
	Name                         string               `json:"name"`
	FullName                     string               `json:"full_name"`
	CloneURL                     string               `json:"clone_url"`
	CloneURLSSH                  string               `json:"clone_url_ssh"`
	ForgeURL                     string               `json:"forge_url"`
	AvatarURL                    string               `json:"avatar_url"`
	DefaultBranch                string               `json:"default_branch"`
	Active                       bool                 `json:"active"`
	Private                      bool                 `json:"private"`
	PREnabled                    bool                 `json:"pr_enabled"`
	AllowDeploy                  bool                 `json:"allow_deploy"`
	AllowManual                  bool                 `json:"allow_manual"`
	AllowPR                      bool                 `json:"allow_pr"`
	Visibility                   string               `json:"visibility"`
	RequireApproval              string               `json:"require_approval"`
	Trusted                      trustedAPIResp       `json:"trusted"`
	ConfigFile                   string               `json:"config_file"`
	DeployTeam                   string               `json:"deploy_team"`
	Timeout                      int64                `json:"timeout"`
	CancelPreviousPipelineEvents []string             `json:"cancel_previous_pipeline_events"`
	NetrcTrusted                 []string             `json:"netrc_trusted"`
	LogsKeepDuration             string               `json:"logs_keep_duration"`
	LogsKeepMin                  int64                `json:"logs_keep_min"`
	LastPipeline                 *pipelineAPIResponse `json:"last_pipeline"`
}

type pipelineAPIResponse struct {
	ID          int64  `json:"id"`
	Number      int64  `json:"number"`
	Status      string `json:"status"`
	Event       string `json:"event"`
	Branch      string `json:"branch"`
	Commit      string `json:"commit"`
	Ref         string `json:"ref"`
	Message     string `json:"message"`
	Author      string `json:"author"`
	AuthorEmail string `json:"author_email"`
	Sender      string `json:"sender"`
	ForgeURL    string `json:"forge_url"`
	Created     int64  `json:"created"`
	Started     int64  `json:"started"`
	Finished    int64  `json:"finished"`
}

func (d *userRepositoriesActiveDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_repositories_active"
}

func (d *userRepositoriesActiveDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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

	lastPipelineAttrs := map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Pipeline ID.",
		},
		"number": schema.Int64Attribute{
			Computed:    true,
			Description: "Pipeline number within the repository.",
		},
		"status": schema.StringAttribute{
			Computed:    true,
			Description: "Pipeline status.",
		},
		"event": schema.StringAttribute{
			Computed:    true,
			Description: "Event that triggered the pipeline.",
		},
		"branch": schema.StringAttribute{
			Computed:    true,
			Description: "Branch the pipeline ran on.",
		},
		"commit": schema.StringAttribute{
			Computed:    true,
			Description: "Commit SHA.",
		},
		"ref": schema.StringAttribute{
			Computed:    true,
			Description: "Git ref.",
		},
		"message": schema.StringAttribute{
			Computed:    true,
			Description: "Commit message.",
		},
		"author": schema.StringAttribute{
			Computed:    true,
			Description: "Commit author name.",
		},
		"author_email": schema.StringAttribute{
			Computed:    true,
			Description: "Commit author email.",
		},
		"sender": schema.StringAttribute{
			Computed:    true,
			Description: "User who triggered the pipeline.",
		},
		"forge_url": schema.StringAttribute{
			Computed:    true,
			Description: "URL to the pipeline at the forge.",
		},
		"created": schema.Int64Attribute{
			Computed:    true,
			Description: "Pipeline creation time as a Unix timestamp.",
		},
		"started": schema.Int64Attribute{
			Computed:    true,
			Description: "Pipeline start time as a Unix timestamp.",
		},
		"finished": schema.Int64Attribute{
			Computed:    true,
			Description: "Pipeline finish time as a Unix timestamp.",
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
		"last_pipeline": schema.SingleNestedAttribute{
			Computed:    true,
			Description: "Last pipeline run for the repository. Null if no pipeline has run yet.",
			Attributes:  lastPipelineAttrs,
		},
	}

	resp.Schema = schema.Schema{
		Description: "Fetches all active repositories for the authenticated user with their last pipeline info.",
		Attributes: map[string]schema.Attribute{
			"repositories": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: repoAttrs,
				},
			},
		},
	}
}

func (d *userRepositoriesActiveDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userRepositoriesActiveDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	endpoint := fmt.Sprintf("%s/api/v1/user/repos/active", d.client.Host)
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
			fmt.Sprintf("GET /user/repos/active returned status %d", httpResp.StatusCode),
		)
		return
	}

	var results []repoLastPipelineAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&results); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	repos := make([]activeRepositoryItemModel, len(results))
	for i, r := range results {
		repos[i] = mapRepoLastPipelineToItem(&r)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &userRepositoriesActiveDataSourceModel{Repositories: repos})...)
}

func mapRepoLastPipelineToItem(r *repoLastPipelineAPIResponse) activeRepositoryItemModel {
	trusted, _ := types.ObjectValue(trustedAttrTypes, map[string]attr.Value{
		"network":  types.BoolValue(r.Trusted.Network),
		"security": types.BoolValue(r.Trusted.Security),
		"volumes":  types.BoolValue(r.Trusted.Volumes),
	})

	var lastPipeline types.Object
	if r.LastPipeline != nil {
		lastPipeline, _ = types.ObjectValue(lastPipelineAttrTypes, map[string]attr.Value{
			"id":           types.Int64Value(r.LastPipeline.ID),
			"number":       types.Int64Value(r.LastPipeline.Number),
			"status":       types.StringValue(r.LastPipeline.Status),
			"event":        types.StringValue(r.LastPipeline.Event),
			"branch":       types.StringValue(r.LastPipeline.Branch),
			"commit":       types.StringValue(r.LastPipeline.Commit),
			"ref":          types.StringValue(r.LastPipeline.Ref),
			"message":      types.StringValue(r.LastPipeline.Message),
			"author":       types.StringValue(r.LastPipeline.Author),
			"author_email": types.StringValue(r.LastPipeline.AuthorEmail),
			"sender":       types.StringValue(r.LastPipeline.Sender),
			"forge_url":    types.StringValue(r.LastPipeline.ForgeURL),
			"created":      types.Int64Value(r.LastPipeline.Created),
			"started":      types.Int64Value(r.LastPipeline.Started),
			"finished":     types.Int64Value(r.LastPipeline.Finished),
		})
	} else {
		lastPipeline = types.ObjectNull(lastPipelineAttrTypes)
	}

	return activeRepositoryItemModel{
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
		LastPipeline:                 lastPipeline,
	}
}
