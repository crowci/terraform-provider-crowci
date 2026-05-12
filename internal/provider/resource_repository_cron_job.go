package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*repositoryCronJobResource)(nil)
var _ resource.ResourceWithConfigure = (*repositoryCronJobResource)(nil)
var _ resource.ResourceWithImportState = (*repositoryCronJobResource)(nil)
var _ resource.ResourceWithIdentity = (*repositoryCronJobResource)(nil)

func NewRepositoryCronJobResource() resource.Resource {
	return &repositoryCronJobResource{}
}

type repositoryCronJobIdentityModel struct {
	RepoID types.Int64 `tfsdk:"repo_id"`
	ID     types.Int64 `tfsdk:"id"`
}

func (r *repositoryCronJobResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"repo_id": identityschema.Int64Attribute{
				RequiredForImport: true,
				Description:       "ID of the repository this cron job belongs to.",
			},
			"id": identityschema.Int64Attribute{
				RequiredForImport: true,
				Description:       "Cron job ID.",
			},
		},
	}
}

type repositoryCronJobResource struct {
	client *crowciClient
}

type repositoryCronJobResourceModel struct {
	// create-only inputs
	RepoID types.Int64 `tfsdk:"repo_id"`
	// required inputs
	Name     types.String `tfsdk:"name"`
	Schedule types.String `tfsdk:"schedule"`
	// optional input
	Branch types.String `tfsdk:"branch"`
	// computed
	ID        types.Int64  `tfsdk:"id"`
	CreatorID types.Int64  `tfsdk:"creator_id"`
	NextExec  types.Int64  `tfsdk:"next_exec"`
	Created   types.Int64  `tfsdk:"created"`
	FailCount types.Int64  `tfsdk:"fail_count"`
	FailMsg   types.String `tfsdk:"fail_msg"`
	Disabled  types.Bool   `tfsdk:"disabled"`
}

type cronJobAPIResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	RepoID    int64  `json:"repo_id"`
	CreatorID int64  `json:"creator_id"`
	Branch    string `json:"branch"`
	Schedule  string `json:"schedule"`
	Created   int64  `json:"created"`
	NextExec  int64  `json:"next_exec"`
	FailCount int64  `json:"fail_count"`
	FailMsg   string `json:"fail_msg"`
	Disabled  bool   `json:"disabled"`
}

func (r *repositoryCronJobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_cron_job"
}

func (r *repositoryCronJobResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a cron job for a repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository this cron job belongs to.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Cron job name.",
			},
			"schedule": schema.StringAttribute{
				Required:    true,
				Description: "Cron schedule expression (e.g. '@daily', '0 0 * * *').",
			},
			"branch": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Branch to run the cron job on. Defaults to the repository's default branch.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Cron job ID assigned by Crow CI.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"creator_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user who created this cron job.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"next_exec": schema.Int64Attribute{
				Computed:    true,
				Description: "Next scheduled execution time as a Unix timestamp.",
			},
			"created": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation time as a Unix timestamp.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"fail_count": schema.Int64Attribute{
				Computed:    true,
				Description: "Number of consecutive failures.",
			},
			"fail_msg": schema.StringAttribute{
				Computed:    true,
				Description: "Last failure message.",
			},
			"disabled": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the cron job is disabled.",
			},
		},
	}
}

func (r *repositoryCronJobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = client
}

func (r *repositoryCronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data repositoryCronJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Branch   string `json:"branch,omitempty"`
	}{
		Name:     data.Name.ValueString(),
		Schedule: data.Schedule.ValueString(),
	}
	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		body.Branch = data.Branch.ValueString()
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/cron", r.client.Host, data.RepoID.ValueInt64())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		resp.Diagnostics.AddError("Failed to build request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.HTTPClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("API request failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("POST /repos/%d/cron returned status %d: %s", data.RepoID.ValueInt64(), httpResp.StatusCode, b),
		)
		return
	}

	var result cronJobAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	mapCronJobToState(&result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryCronJobIdentityModel{RepoID: data.RepoID, ID: data.ID})...)
}

func (r *repositoryCronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data repositoryCronJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/cron/%d", r.client.Host, data.RepoID.ValueInt64(), data.ID.ValueInt64())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build request", err.Error())
		return
	}

	httpResp, err := r.client.HTTPClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("API request failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}
	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("GET /repos/%d/cron/%d returned status %d", data.RepoID.ValueInt64(), data.ID.ValueInt64(), httpResp.StatusCode),
		)
		return
	}

	var result cronJobAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	mapCronJobToState(&result, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryCronJobIdentityModel{RepoID: data.RepoID, ID: data.ID})...)
}

func (r *repositoryCronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryCronJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state repositoryCronJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := struct {
		Name     *string `json:"name,omitempty"`
		Schedule *string `json:"schedule,omitempty"`
		Branch   *string `json:"branch,omitempty"`
	}{}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		body.Name = &v
	}
	if !plan.Schedule.IsNull() && !plan.Schedule.IsUnknown() {
		v := plan.Schedule.ValueString()
		body.Schedule = &v
	}
	if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
		v := plan.Branch.ValueString()
		body.Branch = &v
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/cron/%d", r.client.Host, state.RepoID.ValueInt64(), state.ID.ValueInt64())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(bodyJSON))
	if err != nil {
		resp.Diagnostics.AddError("Failed to build request", err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := r.client.HTTPClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("API request failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(httpResp.Body)
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("PATCH /repos/%d/cron/%d returned status %d: %s", state.RepoID.ValueInt64(), state.ID.ValueInt64(), httpResp.StatusCode, b),
		)
		return
	}

	var result cronJobAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	mapCronJobToState(&result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryCronJobIdentityModel{RepoID: plan.RepoID, ID: plan.ID})...)
}

func (r *repositoryCronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data repositoryCronJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/cron/%d", r.client.Host, data.RepoID.ValueInt64(), data.ID.ValueInt64())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build request", err.Error())
		return
	}

	httpResp, err := r.client.HTTPClient.Do(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("API request failed", err.Error())
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusNoContent && httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("DELETE /repos/%d/cron/%d returned status %d", data.RepoID.ValueInt64(), data.ID.ValueInt64(), httpResp.StatusCode),
		)
	}
}

// ImportState accepts "repo_id/cron_id".
func (r *repositoryCronJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format \"<repo_id>/<cron_id>\", got %q", req.ID),
		)
		return
	}

	repoID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("repo_id must be numeric, got %q: %s", parts[0], err),
		)
		return
	}
	cronID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("cron_id must be numeric, got %q: %s", parts[1], err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repo_id"), repoID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), cronID)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryCronJobIdentityModel{
		RepoID: types.Int64Value(repoID),
		ID:     types.Int64Value(cronID),
	})...)
}

func mapCronJobToState(r *cronJobAPIResponse, data *repositoryCronJobResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.RepoID = types.Int64Value(r.RepoID)
	data.Name = types.StringValue(r.Name)
	data.Schedule = types.StringValue(r.Schedule)
	data.Branch = types.StringValue(r.Branch)
	data.CreatorID = types.Int64Value(r.CreatorID)
	data.NextExec = types.Int64Value(r.NextExec)
	data.Created = types.Int64Value(r.Created)
	data.FailCount = types.Int64Value(r.FailCount)
	data.FailMsg = types.StringValue(r.FailMsg)
	data.Disabled = types.BoolValue(r.Disabled)
}
