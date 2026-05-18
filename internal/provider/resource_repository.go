package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var trustedAttrTypes = map[string]attr.Type{
	"network":  types.BoolType,
	"security": types.BoolType,
	"volumes":  types.BoolType,
}

var _ resource.Resource = (*repositoryResource)(nil)
var _ resource.ResourceWithConfigure = (*repositoryResource)(nil)
var _ resource.ResourceWithImportState = (*repositoryResource)(nil)

func NewRepositoryResource() resource.Resource {
	return &repositoryResource{}
}

type repositoryResource struct {
	resourceWithClient
}

type repositoryResourceModel struct {
	// create-only inputs
	ForgeRemoteID types.String `tfsdk:"forge_remote_id"`
	ForgeID       types.Int64  `tfsdk:"forge_id"`
	// updatable optional+computed
	AllowDeploy                  types.Bool          `tfsdk:"allow_deploy"`
	AllowManual                  types.Bool          `tfsdk:"allow_manual"`
	AllowPR                      types.Bool          `tfsdk:"allow_pr"`
	CancelPreviousPipelineEvents types.List          `tfsdk:"cancel_previous_pipeline_events"`
	ConfigFile                   types.String        `tfsdk:"config_file"`
	DeployTeam                   types.String        `tfsdk:"deploy_team"`
	LogsKeepDuration             types.String        `tfsdk:"logs_keep_duration"`
	LogsKeepMin                  types.Int64         `tfsdk:"logs_keep_min"`
	NetrcTrusted                 types.List          `tfsdk:"netrc_trusted"`
	RequireApproval              types.String        `tfsdk:"require_approval"`
	Timeout                      types.Int64         `tfsdk:"timeout"`
	Trusted                      types.Object        `tfsdk:"trusted"`
	Visibility                   types.String        `tfsdk:"visibility"`
	// computed only
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	FullName      types.String `tfsdk:"full_name"`
	Owner         types.String `tfsdk:"owner"`
	OrgID         types.Int64  `tfsdk:"org_id"`
	CloneURL      types.String `tfsdk:"clone_url"`
	CloneURLSSH   types.String `tfsdk:"clone_url_ssh"`
	ForgeURL      types.String `tfsdk:"forge_url"`
	AvatarURL     types.String `tfsdk:"avatar_url"`
	DefaultBranch types.String `tfsdk:"default_branch"`
	Private       types.Bool   `tfsdk:"private"`
	PREnabled     types.Bool   `tfsdk:"pr_enabled"`
	Active        types.Bool   `tfsdk:"active"`
}

type repositoryAPIResponse struct {
	ID                           int64          `json:"id"`
	ForgeRemoteID                string         `json:"forge_remote_id"`
	ForgeID                      int64          `json:"forge_id"`
	OrgID                        int64          `json:"org_id"`
	Owner                        string         `json:"owner"`
	Name                         string         `json:"name"`
	FullName                     string         `json:"full_name"`
	CloneURL                     string         `json:"clone_url"`
	CloneURLSSH                  string         `json:"clone_url_ssh"`
	ForgeURL                     string         `json:"forge_url"`
	AvatarURL                    string         `json:"avatar_url"`
	DefaultBranch                string         `json:"default_branch"`
	Active                       bool           `json:"active"`
	Private                      bool           `json:"private"`
	PREnabled                    bool           `json:"pr_enabled"`
	AllowDeploy                  bool           `json:"allow_deploy"`
	AllowManual                  bool           `json:"allow_manual"`
	AllowPR                      bool           `json:"allow_pr"`
	Visibility                   string         `json:"visibility"`
	RequireApproval              string         `json:"require_approval"`
	Trusted                      trustedAPIResp `json:"trusted"`
	ConfigFile                   string         `json:"config_file"`
	DeployTeam                   string         `json:"deploy_team"`
	Timeout                      int64          `json:"timeout"`
	CancelPreviousPipelineEvents []string       `json:"cancel_previous_pipeline_events"`
	NetrcTrusted                 []string       `json:"netrc_trusted"`
	LogsKeepDuration             string         `json:"logs_keep_duration"`
	LogsKeepMin                  int64          `json:"logs_keep_min"`
}

type trustedAPIResp struct {
	Network  bool `json:"network"`
	Security bool `json:"security"`
	Volumes  bool `json:"volumes"`
}

func (r *repositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *repositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Activates a repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"forge_remote_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the repository at the forge.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"forge_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Forge ID (for multi-forge setups). Uses the default forge when not set.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"allow_deploy": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow deploy pipelines.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_manual": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow manual pipeline triggers.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"allow_pr": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow pull request pipelines.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"cancel_previous_pipeline_events": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Pipeline events for which previous runs should be cancelled.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"config_file": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Path to the pipeline configuration file.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deploy_team": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Team allowed to trigger deploy pipelines.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"logs_keep_duration": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Duration to keep pipeline logs (e.g. 720h).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"logs_keep_min": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Minimum number of pipeline logs to keep.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"netrc_trusted": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Hostnames for which .netrc credentials are trusted.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"require_approval": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Approval requirement for pipelines ('forks', 'all', 'none').",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Pipeline execution timeout in minutes.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"trusted": schema.SingleNestedAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Trusted configuration for the repository.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"network": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Allow network access for trusted operations.",
					},
					"security": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Allow security-sensitive operations.",
					},
					"volumes": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Description: "Allow volume mounts.",
					},
				},
			},
			"visibility": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Repository visibility ('public', 'private', 'internal').",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Repository ID assigned by Crow CI.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Repository name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"full_name": schema.StringAttribute{
				Computed:    true,
				Description: "Full repository name (owner/name).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.StringAttribute{
				Computed:    true,
				Description: "Repository owner.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Organization ID the repository belongs to.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"clone_url": schema.StringAttribute{
				Computed:    true,
				Description: "HTTPS clone URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"clone_url_ssh": schema.StringAttribute{
				Computed:    true,
				Description: "SSH clone URL.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forge_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL of the repository at the forge.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"avatar_url": schema.StringAttribute{
				Computed:    true,
				Description: "Avatar URL for the repository.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"default_branch": schema.StringAttribute{
				Computed:    true,
				Description: "Default branch of the repository.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository is private.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"pr_enabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether pull request pipelines are enabled.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"active": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository is active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *repositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data repositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := url.Values{}
	params.Set("forge_remote_id", data.ForgeRemoteID.ValueString())
	if !data.ForgeID.IsNull() && !data.ForgeID.IsUnknown() {
		params.Set("forge_id", strconv.FormatInt(data.ForgeID.ValueInt64(), 10))
	}

	postEndpoint := fmt.Sprintf("%s/api/v1/repos?%s", r.client.Host, params.Encode())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPost, postEndpoint, nil, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var createResult repositoryAPIResponse
	if !decodeJSON(httpResp.Body, &createResult, &resp.Diagnostics) { return }

	// PATCH to apply any additional planned attributes (timeout, visibility, etc.)
	// that the POST endpoint does not accept.
	patchBody := buildRepoPatchBody(ctx, &data)
	patchJSON := marshalJSON(patchBody, &resp.Diagnostics)
	if patchJSON == nil { return }

	patchEndpoint := fmt.Sprintf("%s/api/v1/repos/%d", r.client.Host, createResult.ID)
	patchResp, ok := doRequest(ctx, r.client, http.MethodPatch, patchEndpoint, patchJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer patchResp.Body.Close()

	var result repositoryAPIResponse
	if !decodeJSON(patchResp.Body, &result, &resp.Diagnostics) { return }

	mapRepoToState(&result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *repositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data repositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d", r.client.Host, data.ID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var result repositoryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	mapRepoToState(&result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *repositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state repositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildRepoPatchBody(ctx, &plan)
	bodyJSON := marshalJSON(body, &resp.Diagnostics)
	if bodyJSON == nil { return }

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d", r.client.Host, state.ID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPatch, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result repositoryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	mapRepoToState(&result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *repositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data repositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d", r.client.Host, data.ID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodDelete, endpoint, nil, []int{http.StatusNoContent, http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	httpResp.Body.Close()
}

func (r *repositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected a numeric repository ID, got %q: %s", req.ID, err),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

type repoPatchBody struct {
	AllowDeploy                  *bool             `json:"allow_deploy,omitempty"`
	AllowManual                  *bool             `json:"allow_manual,omitempty"`
	AllowPR                      *bool             `json:"allow_pr,omitempty"`
	CancelPreviousPipelineEvents []string          `json:"cancel_previous_pipeline_events,omitempty"`
	ConfigFile                   *string           `json:"config_file,omitempty"`
	DeployTeam                   *string           `json:"deploy_team,omitempty"`
	LogsKeepDuration             *string           `json:"logs_keep_duration,omitempty"`
	LogsKeepMin                  *int64            `json:"logs_keep_min,omitempty"`
	NetrcTrusted                 []string          `json:"netrc_trusted,omitempty"`
	RequireApproval              *string           `json:"require_approval,omitempty"`
	Timeout                      *int64            `json:"timeout,omitempty"`
	Trusted                      *trustedPatchBody `json:"trusted,omitempty"`
	Visibility                   *string           `json:"visibility,omitempty"`
}

type trustedPatchBody struct {
	Network  *bool `json:"network,omitempty"`
	Security *bool `json:"security,omitempty"`
	Volumes  *bool `json:"volumes,omitempty"`
}

func buildRepoPatchBody(ctx context.Context, plan *repositoryResourceModel) repoPatchBody {
	body := repoPatchBody{}

	if !plan.AllowDeploy.IsNull() && !plan.AllowDeploy.IsUnknown() {
		v := plan.AllowDeploy.ValueBool()
		body.AllowDeploy = &v
	}
	if !plan.AllowManual.IsNull() && !plan.AllowManual.IsUnknown() {
		v := plan.AllowManual.ValueBool()
		body.AllowManual = &v
	}
	if !plan.AllowPR.IsNull() && !plan.AllowPR.IsUnknown() {
		v := plan.AllowPR.ValueBool()
		body.AllowPR = &v
	}
	if !plan.CancelPreviousPipelineEvents.IsNull() && !plan.CancelPreviousPipelineEvents.IsUnknown() {
		body.CancelPreviousPipelineEvents = listToStrings(plan.CancelPreviousPipelineEvents)
	}
	if !plan.ConfigFile.IsNull() && !plan.ConfigFile.IsUnknown() {
		v := plan.ConfigFile.ValueString()
		body.ConfigFile = &v
	}
	if !plan.DeployTeam.IsNull() && !plan.DeployTeam.IsUnknown() {
		v := plan.DeployTeam.ValueString()
		body.DeployTeam = &v
	}
	if !plan.LogsKeepDuration.IsNull() && !plan.LogsKeepDuration.IsUnknown() {
		v := plan.LogsKeepDuration.ValueString()
		body.LogsKeepDuration = &v
	}
	if !plan.LogsKeepMin.IsNull() && !plan.LogsKeepMin.IsUnknown() {
		v := plan.LogsKeepMin.ValueInt64()
		body.LogsKeepMin = &v
	}
	if !plan.NetrcTrusted.IsNull() && !plan.NetrcTrusted.IsUnknown() {
		body.NetrcTrusted = listToStrings(plan.NetrcTrusted)
	}
	if !plan.RequireApproval.IsNull() && !plan.RequireApproval.IsUnknown() {
		v := plan.RequireApproval.ValueString()
		body.RequireApproval = &v
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		v := plan.Timeout.ValueInt64()
		body.Timeout = &v
	}
	if !plan.Trusted.IsNull() && !plan.Trusted.IsUnknown() {
		tp := &trustedPatchBody{}
		attrs := plan.Trusted.Attributes()
		if v, ok := attrs["network"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
			b := v.ValueBool()
			tp.Network = &b
		}
		if v, ok := attrs["security"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
			b := v.ValueBool()
			tp.Security = &b
		}
		if v, ok := attrs["volumes"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
			b := v.ValueBool()
			tp.Volumes = &b
		}
		body.Trusted = tp
	}
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		v := plan.Visibility.ValueString()
		body.Visibility = &v
	}

	return body
}

func mapRepoToState(r *repositoryAPIResponse, data *repositoryResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.ForgeRemoteID = types.StringValue(r.ForgeRemoteID)
	data.ForgeID = types.Int64Value(r.ForgeID)
	data.OrgID = types.Int64Value(r.OrgID)
	data.Owner = types.StringValue(r.Owner)
	data.Name = types.StringValue(r.Name)
	data.FullName = types.StringValue(r.FullName)
	data.CloneURL = types.StringValue(r.CloneURL)
	data.CloneURLSSH = types.StringValue(r.CloneURLSSH)
	data.ForgeURL = types.StringValue(r.ForgeURL)
	data.AvatarURL = types.StringValue(r.AvatarURL)
	data.DefaultBranch = types.StringValue(r.DefaultBranch)
	data.Active = types.BoolValue(r.Active)
	data.Private = types.BoolValue(r.Private)
	data.PREnabled = types.BoolValue(r.PREnabled)
	data.AllowDeploy = types.BoolValue(r.AllowDeploy)
	data.AllowManual = types.BoolValue(r.AllowManual)
	data.AllowPR = types.BoolValue(r.AllowPR)
	data.Visibility = types.StringValue(r.Visibility)
	data.RequireApproval = types.StringValue(r.RequireApproval)
	data.ConfigFile = types.StringValue(r.ConfigFile)
	data.DeployTeam = types.StringValue(r.DeployTeam)
	data.Timeout = types.Int64Value(r.Timeout)
	data.CancelPreviousPipelineEvents = stringsToList(r.CancelPreviousPipelineEvents)
	data.NetrcTrusted = stringsToList(r.NetrcTrusted)
	data.LogsKeepDuration = types.StringValue(r.LogsKeepDuration)
	data.LogsKeepMin = types.Int64Value(r.LogsKeepMin)
	data.Trusted, _ = types.ObjectValue(trustedAttrTypes, map[string]attr.Value{
		"network":  types.BoolValue(r.Trusted.Network),
		"security": types.BoolValue(r.Trusted.Security),
		"volumes":  types.BoolValue(r.Trusted.Volumes),
	})
}
