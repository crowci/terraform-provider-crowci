package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*userAccessTokenResource)(nil)
var _ resource.ResourceWithConfigure = (*userAccessTokenResource)(nil)
var _ resource.ResourceWithImportState = (*userAccessTokenResource)(nil)

func NewUserAccessTokenResource() resource.Resource {
	return &userAccessTokenResource{}
}

type userAccessTokenResource struct {
	resourceWithClient
}

type userAccessTokenResourceModel struct {
	// required inputs
	Name   types.String `tfsdk:"name"`
	Scopes types.List   `tfsdk:"scopes"`
	// optional create-only inputs
	ExpiresAt types.Int64 `tfsdk:"expires_at"`
	OrgID     types.Int64 `tfsdk:"org_id"`
	RepoID    types.Int64 `tfsdk:"repo_id"`
	// computed
	ID        types.Int64  `tfsdk:"id"`
	Token     types.String `tfsdk:"token"`
	UserID    types.Int64  `tfsdk:"user_id"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
	LastUsed  types.Int64  `tfsdk:"last_used"`
}

func (r *userAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_access_token"
}

func (r *userAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Access token for authentication using API to Crow server.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Token name.",
			},
			"scopes": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Token scopes.",
			},
			"expires_at": schema.Int64Attribute{
				Optional:    true,
				Description: "Optional expiry as a Unix timestamp.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Optional org scope.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"repo_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Optional repo scope.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Token ID assigned by Crow CI.",
			},
			"token": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The token secret. Only populated on creation; not retrievable afterwards.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "ID of the user who owns this token.",
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation time as Unix timestamp.",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Last update time as Unix timestamp.",
			},
			"last_used": schema.Int64Attribute{
				Computed:    true,
				Description: "Last use time as Unix timestamp.",
			},
		},
	}
}

func (r *userAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data userAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes := listToStrings(data.Scopes)

	body := accessTokenCreateRequest{
		Name:   data.Name.ValueString(),
		Scopes: scopes,
	}
	if !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		v := data.ExpiresAt.ValueInt64()
		body.ExpiresAt = &v
	}
	if !data.OrgID.IsNull() && !data.OrgID.IsUnknown() {
		v := data.OrgID.ValueInt64()
		body.OrgID = &v
	}
	if !data.RepoID.IsNull() && !data.RepoID.IsUnknown() {
		v := data.RepoID.ValueInt64()
		body.RepoID = &v
	}

	bodyJSON := marshalJSON(body, &resp.Diagnostics)
	if bodyJSON == nil { return }

	endpoint := fmt.Sprintf("%s/api/v1/user/access-tokens", r.client.Host)
	httpResp, ok := doRequest(ctx, r.client, http.MethodPost, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result accessTokenAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	mapAccessTokenToState(&result, &data)
	// token is only present in the create response.
	data.Token = types.StringValue(result.Token)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data userAccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/user/access-tokens/%d", r.client.Host, data.ID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	var result accessTokenAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	// Preserve the token value from prior state — GET does not return it.
	token := data.Token
	mapAccessTokenToState(&result, &data)
	data.Token = token

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *userAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userAccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state userAccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID
	plan.Token = state.Token

	scopes := listToStrings(plan.Scopes)
	name := plan.Name.ValueString()
	body := accessTokenUpdateRequest{
		Name:   &name,
		Scopes: scopes,
	}

	bodyJSON := marshalJSON(body, &resp.Diagnostics)
	if bodyJSON == nil { return }

	endpoint := fmt.Sprintf("%s/api/v1/user/access-tokens/%d", r.client.Host, plan.ID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPatch, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result accessTokenAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	mapAccessTokenToState(&result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data userAccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/user/access-tokens/%d", r.client.Host, data.ID.ValueInt64())
	httpResp, ok := doRequest(ctx, r.client, http.MethodDelete, endpoint, nil, []int{http.StatusNoContent, http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	httpResp.Body.Close()
}

func (r *userAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected a numeric token ID, got %q: %s", req.ID, err),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// --- shared types ---

type accessTokenAPIResponse struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	Token     string   `json:"token"` // only present in create response
	UserID    int64    `json:"user_id"`
	OrgID     int64    `json:"org_id"`
	RepoID    int64    `json:"repo_id"`
	ExpiresAt int64    `json:"expires_at"`
	LastUsed  int64    `json:"last_used"`
	CreatedAt int64    `json:"created_at"`
	UpdatedAt int64    `json:"updated_at"`
}

type accessTokenCreateRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *int64   `json:"expires_at,omitempty"`
	OrgID     *int64   `json:"org_id,omitempty"`
	RepoID    *int64   `json:"repo_id,omitempty"`
}

type accessTokenUpdateRequest struct {
	Name   *string  `json:"name,omitempty"`
	Scopes []string `json:"scopes,omitempty"`
}

func mapAccessTokenToState(r *accessTokenAPIResponse, data *userAccessTokenResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.Name = types.StringValue(r.Name)
	data.UserID = types.Int64Value(r.UserID)
	// Optional fields: keep null when the API returns the zero value so that
	// Terraform does not see a planned null become an unexpected 0.
	data.OrgID = int64NullIfZero(r.OrgID)
	data.RepoID = int64NullIfZero(r.RepoID)
	data.ExpiresAt = int64NullIfZero(r.ExpiresAt)
	data.LastUsed = types.Int64Value(r.LastUsed)
	data.CreatedAt = types.Int64Value(r.CreatedAt)
	data.UpdatedAt = types.Int64Value(r.UpdatedAt)

	elems := make([]attr.Value, len(r.Scopes))
	for i, s := range r.Scopes {
		elems[i] = types.StringValue(s)
	}
	data.Scopes, _ = types.ListValue(types.StringType, elems)
}
