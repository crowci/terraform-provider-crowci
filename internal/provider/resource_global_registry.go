package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*globalRegistryResource)(nil)
var _ resource.ResourceWithConfigure = (*globalRegistryResource)(nil)
var _ resource.ResourceWithImportState = (*globalRegistryResource)(nil)

func NewGlobalRegistryResource() resource.Resource {
	return &globalRegistryResource{}
}

type globalRegistryResource struct {
	resourceWithClient
}

type globalRegistryResourceModel struct {
	// required create-only inputs
	Address types.String `tfsdk:"address"`
	// required inputs
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	// computed
	ID        types.Int64  `tfsdk:"id"`
	ReadOnly  types.Bool   `tfsdk:"readonly"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (r *globalRegistryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_registry"
}

func (r *globalRegistryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a global container registry credential on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
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
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
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

func (r *globalRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data globalRegistryResourceModel
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

	endpoint := fmt.Sprintf("%s/api/v1/registries", r.client.Host)
	httpResp, ok := doRequest(ctx, r.client, http.MethodPost, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	password := data.Password
	mapGlobalRegistryToState(&result, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *globalRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data globalRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/registries/%s", r.client.Host, data.Address.ValueString())
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
	mapGlobalRegistryToState(&result, &data)
	data.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *globalRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan globalRegistryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state globalRegistryResourceModel
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

	endpoint := fmt.Sprintf("%s/api/v1/registries/%s", r.client.Host, state.Address.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodPatch, endpoint, bodyJSON, []int{http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	var result registryAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	password := plan.Password
	mapGlobalRegistryToState(&result, &plan)
	plan.Password = password

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *globalRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data globalRegistryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/registries/%s", r.client.Host, data.Address.ValueString())
	httpResp, ok := doRequest(ctx, r.client, http.MethodDelete, endpoint, nil, []int{http.StatusNoContent, http.StatusOK}, &resp.Diagnostics)
	if !ok { return }
	httpResp.Body.Close()
}

// ImportState accepts the registry address as the import ID.
func (r *globalRegistryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("address"), req.ID)...)
}

func mapGlobalRegistryToState(r *registryAPIResponse, data *globalRegistryResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.Address = types.StringValue(r.Address)
	data.Username = types.StringValue(r.Username)
	data.ReadOnly = types.BoolValue(r.ReadOnly)
	data.CreatedAt = types.Int64Value(r.CreatedAt)
	data.UpdatedAt = types.Int64Value(r.UpdatedAt)
}
