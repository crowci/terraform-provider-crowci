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

var _ resource.Resource = (*organizationRegistryResource)(nil)
var _ resource.ResourceWithConfigure = (*organizationRegistryResource)(nil)
var _ resource.ResourceWithImportState = (*organizationRegistryResource)(nil)
var _ resource.ResourceWithIdentity = (*organizationRegistryResource)(nil)

func NewOrganizationRegistryResource() resource.Resource {
	return &organizationRegistryResource{}
}

type organizationRegistryIdentityModel struct {
	OrgID   types.Int64  `tfsdk:"org_id"`
	Address types.String `tfsdk:"address"`
}

func (r *organizationRegistryResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"org_id": identityschema.Int64Attribute{
				RequiredForImport: true,
				Description:       "ID of the organization this registry belongs to.",
			},
			"address": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "Registry address.",
			},
		},
	}
}

type organizationRegistryResource struct {
	resourceWithClient
}

type organizationRegistryResourceModel struct {
	// create-only inputs
	OrgID types.Int64 `tfsdk:"org_id"`
	// required inputs
	Address  types.String `tfsdk:"address"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	// computed
	ID       types.Int64 `tfsdk:"id"`
	ReadOnly types.Bool  `tfsdk:"readonly"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

type registryAPIResponse struct {
	ID        int64  `json:"id"`
	OrgID     int64  `json:"org_id"`
	RepoID    int64  `json:"repo_id"`
	Address   string `json:"address"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ReadOnly  bool   `json:"readonly"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (r *organizationRegistryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_registry"
}

func (r *organizationRegistryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a container registry credential scoped to an organization on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"org_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the organization this registry belongs to.",
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

func (r *organizationRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data organizationRegistryResourceModel
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

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%d/registries", r.client.Host, data.OrgID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPost, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	password := data.Password
	mapOrgRegistryToState(&result, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, organizationRegistryIdentityModel{OrgID: data.OrgID, Address: data.Address})...)
}

func (r *organizationRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data organizationRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%d/registries/%s", r.client.Host, data.OrgID.ValueInt64(), data.Address.ValueString())
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
	mapOrgRegistryToState(&result, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, organizationRegistryIdentityModel{OrgID: data.OrgID, Address: data.Address})...)
}

func (r *organizationRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationRegistryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state organizationRegistryResourceModel
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

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%d/registries/%s", r.client.Host, state.OrgID.ValueInt64(), state.Address.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPatch, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	password := plan.Password
	mapOrgRegistryToState(&result, &plan)
	plan.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, organizationRegistryIdentityModel{OrgID: plan.OrgID, Address: plan.Address})...)
}

func (r *organizationRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data organizationRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/orgs/%d/registries/%s", r.client.Host, data.OrgID.ValueInt64(), data.Address.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodDelete, endpoint, nil, []int{http.StatusNoContent, http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	httpResp.Body.Close()
}

// ImportState accepts "org_id/address".
func (r *organizationRegistryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format \"<org_id>/<address>\", got %q", req.ID),
		)
		return
	}

	var orgID int64
	if _, err := fmt.Sscanf(parts[0], "%d", &orgID); err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("org_id must be numeric, got %q: %s", parts[0], err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("org_id"), orgID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("address"), parts[1])...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, organizationRegistryIdentityModel{
		OrgID:   types.Int64Value(orgID),
		Address: types.StringValue(parts[1]),
	})...)
}

func mapOrgRegistryToState(r *registryAPIResponse, data *organizationRegistryResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.OrgID = types.Int64Value(r.OrgID)
	data.Address = types.StringValue(r.Address)
	data.Username = types.StringValue(r.Username)
	data.ReadOnly = types.BoolValue(r.ReadOnly)
	data.CreatedAt = types.Int64Value(r.CreatedAt)
	data.UpdatedAt = types.Int64Value(r.UpdatedAt)
}
