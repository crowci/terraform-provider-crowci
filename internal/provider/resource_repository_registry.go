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

var _ resource.Resource = (*repositoryRegistryResource)(nil)
var _ resource.ResourceWithConfigure = (*repositoryRegistryResource)(nil)
var _ resource.ResourceWithImportState = (*repositoryRegistryResource)(nil)
var _ resource.ResourceWithIdentity = (*repositoryRegistryResource)(nil)

func NewRepositoryRegistryResource() resource.Resource {
	return &repositoryRegistryResource{}
}

type repositoryRegistryIdentityModel struct {
	RepoID  types.Int64  `tfsdk:"repo_id"`
	Address types.String `tfsdk:"address"`
}

type repositoryRegistryResource struct {
	resourceWithClient
}

type repositoryRegistryResourceModel struct {
	// create-only inputs
	RepoID types.Int64 `tfsdk:"repo_id"`
	// required inputs
	Address  types.String `tfsdk:"address"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	// computed
	ID        types.Int64  `tfsdk:"id"`
	ReadOnly  types.Bool   `tfsdk:"readonly"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (r *repositoryRegistryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_registry"
}

func (r *repositoryRegistryResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"repo_id": identityschema.Int64Attribute{
				RequiredForImport: true,
				Description:       "ID of the repository this registry belongs to.",
			},
			"address": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "Registry address.",
			},
		},
	}
}

func (r *repositoryRegistryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a container registry credential scoped to a repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository this registry belongs to.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"address": schema.StringAttribute{
				Required:    true,
				Description: "Registry address (e.g. 'docker.io', 'ghcr.io').",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "Registry username.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Registry password. Not returned by the API after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Registry ID assigned by Crow CI.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"readonly": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the registry is read-only.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation time as a Unix timestamp.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Last update time as a Unix timestamp.",
			},
		},
	}
}

func (r *repositoryRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data repositoryRegistryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := struct {
		Address  string `json:"address"`
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Address:  data.Address.ValueString(),
		Username: data.Username.ValueString(),
		Password: data.Password.ValueString(),
	}

	bodyJSON := marshalJSON(body, &resp.Diagnostics)
	if bodyJSON == nil { return }

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/registries", r.client.Host, data.RepoID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPost, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	password := data.Password
	mapRepoRegistryToState(&result, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryRegistryIdentityModel{RepoID: data.RepoID, Address: data.Address})...)
}

func (r *repositoryRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data repositoryRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/registries/%s", r.client.Host, data.RepoID.ValueInt64(), data.Address.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	password := data.Password
	mapRepoRegistryToState(&result, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryRegistryIdentityModel{RepoID: data.RepoID, Address: data.Address})...)
}

func (r *repositoryRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryRegistryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state repositoryRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	body := struct {
		Username *string `json:"username,omitempty"`
		Password *string `json:"password,omitempty"`
	}{}
	if !plan.Username.IsNull() && !plan.Username.IsUnknown() {
		v := plan.Username.ValueString()
		body.Username = &v
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		v := plan.Password.ValueString()
		body.Password = &v
	}

	bodyJSON := marshalJSON(body, &resp.Diagnostics)
	if bodyJSON == nil { return }

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/registries/%s", r.client.Host, state.RepoID.ValueInt64(), state.Address.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPatch, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	password := plan.Password
	mapRepoRegistryToState(&result, &plan)
	plan.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryRegistryIdentityModel{RepoID: plan.RepoID, Address: plan.Address})...)
}

func (r *repositoryRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data repositoryRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/registries/%s", r.client.Host, data.RepoID.ValueInt64(), data.Address.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodDelete, endpoint, nil, []int{http.StatusNoContent, http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	httpResp.Body.Close()
}

// ImportState accepts "repo_id/address".
func (r *repositoryRegistryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format \"<repo_id>/<address>\", got %q", req.ID),
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
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("address"), parts[1])...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, repositoryRegistryIdentityModel{
		RepoID:  types.Int64Value(repoID),
		Address: types.StringValue(parts[1]),
	})...)
}

func mapRepoRegistryToState(r *registryAPIResponse, data *repositoryRegistryResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.RepoID = types.Int64Value(r.RepoID)
	data.Address = types.StringValue(r.Address)
	data.Username = types.StringValue(r.Username)
	data.ReadOnly = types.BoolValue(r.ReadOnly)
	data.CreatedAt = types.Int64Value(r.CreatedAt)
	data.UpdatedAt = types.Int64Value(r.UpdatedAt)
}
