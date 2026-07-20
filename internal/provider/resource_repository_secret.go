package provider

import (
	"context"
	"fmt"
	"net/http"
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

var _ resource.Resource = (*repositorySecretResource)(nil)
var _ resource.ResourceWithConfigure = (*repositorySecretResource)(nil)
var _ resource.ResourceWithImportState = (*repositorySecretResource)(nil)
var _ resource.ResourceWithIdentity = (*repositorySecretResource)(nil)

func NewRepositorySecretResource() resource.Resource {
	return &repositorySecretResource{}
}

type repositorySecretResource struct {
	resourceWithClient
}

type repositorySecretIdentityModel struct {
	RepoID types.Int64  `tfsdk:"repo_id"`
	Name   types.String `tfsdk:"name"`
}

type repositorySecretResourceModel struct {
	// required create-only inputs
	RepoID types.Int64  `tfsdk:"repo_id"`
	Name   types.String `tfsdk:"name"`
	// required inputs
	Value  types.String `tfsdk:"value"`
	Events types.List   `tfsdk:"events"`
	// optional inputs
	Images types.List `tfsdk:"images"`
	// computed
	ID        types.Int64  `tfsdk:"id"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (r *repositorySecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_secret"
}

func (r *repositorySecretResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"repo_id": identityschema.Int64Attribute{
				RequiredForImport: true,
				Description:       "ID of the repository this secret belongs to.",
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "Secret name.",
			},
		},
	}
}

func (r *repositorySecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Secret scoped to a specific repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository this secret belongs to.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Secret name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Secret value. Not returned by the API after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"events": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Events that trigger the secret.",
			},
			"images": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Container images the secret is available to. Empty list means all images.",
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Secret ID assigned by Crow CI.",
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

func (r *repositorySecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data repositorySecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := struct {
		Name   string   `json:"name"`
		Value  string   `json:"value"`
		Events []string `json:"events"`
		Images []string `json:"images,omitempty"`
	}{
		Name:   data.Name.ValueString(),
		Value:  data.Value.ValueString(),
		Events: listToStrings(data.Events),
		Images: listToStrings(data.Images),
	}

	bodyJSON := marshalJSON(body, &resp.Diagnostics)
	if bodyJSON == nil {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/secrets", r.client.Host, data.RepoID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPost, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok {
		return
	}
	defer httpResp.Body.Close()

	var result globalSecretAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) {
		return
	}

	value := data.Value
	mapRepoSecretToState(&result, &data)
	data.Value = value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositorySecretIdentityModel{RepoID: data.RepoID, Name: data.Name})...)
}

func (r *repositorySecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data repositorySecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/secrets/%s", r.client.Host, data.RepoID.ValueInt64(), data.Name.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok {
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var result globalSecretAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) {
		return
	}

	// value is not returned by the API — preserve from prior state.
	value := data.Value
	mapRepoSecretToState(&result, &data)
	data.Value = value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositorySecretIdentityModel{RepoID: data.RepoID, Name: data.Name})...)
}

func (r *repositorySecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositorySecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state repositorySecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	body := struct {
		Value  string   `json:"value,omitempty"`
		Events []string `json:"events,omitempty"`
		Images []string `json:"images,omitempty"`
	}{
		Value:  plan.Value.ValueString(),
		Events: listToStrings(plan.Events),
		Images: listToStrings(plan.Images),
	}

	bodyJSON := marshalJSON(body, &resp.Diagnostics)
	if bodyJSON == nil {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/secrets/%s", r.client.Host, plan.RepoID.ValueInt64(), plan.Name.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPatch, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok {
		return
	}
	defer httpResp.Body.Close()

	var result globalSecretAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) {
		return
	}

	savedValue := plan.Value
	mapRepoSecretToState(&result, &plan)
	plan.Value = savedValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositorySecretIdentityModel{RepoID: plan.RepoID, Name: plan.Name})...)
}

func (r *repositorySecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data repositorySecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/secrets/%s", r.client.Host, data.RepoID.ValueInt64(), data.Name.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodDelete, endpoint, nil, []int{http.StatusNoContent, http.StatusOK}, &resp.Diagnostics)
	if !ok {
		return
	}
	httpResp.Body.Close()
}

// ImportState accepts "repo_id/secret_name".
func (r *repositorySecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format \"<repo_id>/<secret_name>\", got %q", req.ID),
		)
		return
	}

	var repoID int64
	if _, err := fmt.Sscanf(parts[0], "%d", &repoID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("repo_id must be numeric, got %q: %s", parts[0], err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repo_id"), repoID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositorySecretIdentityModel{
		RepoID: types.Int64Value(repoID),
		Name:   types.StringValue(parts[1]),
	})...)
}

func mapRepoSecretToState(r *globalSecretAPIResponse, data *repositorySecretResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.Name = types.StringValue(r.Name)
	data.RepoID = int64NullIfZero(r.RepoID)
	data.Source = types.StringValue(r.Source)
	data.CreatedAt = types.Int64Value(r.CreatedAt)
	data.UpdatedAt = types.Int64Value(r.UpdatedAt)
	data.Events = stringsToList(r.Events)
	data.Images = stringsToList(r.Images)
}
