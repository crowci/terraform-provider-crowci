package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = (*globalSecretResource)(nil)
var _ resource.ResourceWithConfigure = (*globalSecretResource)(nil)
var _ resource.ResourceWithImportState = (*globalSecretResource)(nil)

func NewGlobalSecretResource() resource.Resource {
	return &globalSecretResource{}
}

type globalSecretResource struct {
	client *crowciClient
}

type globalSecretResourceModel struct {
	// required inputs
	Name   types.String `tfsdk:"name"`
	Value  types.String `tfsdk:"value"`
	Events types.List   `tfsdk:"events"`
	// optional inputs
	Images types.List `tfsdk:"images"`
	// computed
	ID        types.Int64  `tfsdk:"id"`
	OrgID     types.Int64  `tfsdk:"org_id"`
	RepoID    types.Int64  `tfsdk:"repo_id"`
	Source    types.String `tfsdk:"source"`
	CreatedAt types.Int64  `tfsdk:"created_at"`
	UpdatedAt types.Int64  `tfsdk:"updated_at"`
}

func (r *globalSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_secret"
}

func (r *globalSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Global secret available to all repositories on the Crow CI server. " +
			"Note: If you have no secret yet and you try to create multple global secret, you will encounter an error. " +
			"A workaround is either you rerun the apply, or create an empty secret by hand or code using `depends_on` block.",
		Attributes: map[string]schema.Attribute{
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
			"org_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Org scope of the secret.",
			},
			"repo_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Repo scope of the secret.",
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

func (r *globalSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *globalSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data globalSecretResourceModel
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
		Events: listToStrings(ctx, data.Events),
		Images: listToStrings(ctx, data.Images),
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/secrets", r.client.Host)
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
			fmt.Sprintf("POST /secrets returned status %d: %s", httpResp.StatusCode, b),
		)
		return
	}

	var result globalSecretAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	value := data.Value
	mapGlobalSecretToState(&result, &data)
	data.Value = value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *globalSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data globalSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/secrets/%s", r.client.Host, data.Name.ValueString())
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
			fmt.Sprintf("GET /secrets/%s returned status %d", data.Name.ValueString(), httpResp.StatusCode),
		)
		return
	}

	var result globalSecretAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	// value is not returned by the API — preserve from prior state.
	value := data.Value
	mapGlobalSecretToState(&result, &data)
	data.Value = value

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *globalSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan globalSecretResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state globalSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = state.ID

	events := listToStrings(ctx, plan.Events)
	images := listToStrings(ctx, plan.Images)
	value := plan.Value.ValueString()

	body := struct {
		Value  string   `json:"value,omitempty"`
		Events []string `json:"events,omitempty"`
		Images []string `json:"images,omitempty"`
	}{
		Value:  value,
		Events: events,
		Images: images,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to encode request", err.Error())
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/secrets/%s", r.client.Host, plan.Name.ValueString())
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
		resp.Diagnostics.AddError(
			"Unexpected API response",
			fmt.Sprintf("PATCH /secrets/%s returned status %d", plan.Name.ValueString(), httpResp.StatusCode),
		)
		return
	}

	var result globalSecretAPIResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		resp.Diagnostics.AddError("Failed to decode response", err.Error())
		return
	}

	savedValue := plan.Value
	mapGlobalSecretToState(&result, &plan)
	plan.Value = savedValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *globalSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data globalSecretResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/secrets/%s", r.client.Host, data.Name.ValueString())
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
			fmt.Sprintf("DELETE /secrets/%s returned status %d", data.Name.ValueString(), httpResp.StatusCode),
		)
	}
}

func (r *globalSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

func mapGlobalSecretToState(r *globalSecretAPIResponse, data *globalSecretResourceModel) {
	data.ID = types.Int64Value(r.ID)
	data.Name = types.StringValue(r.Name)
	data.OrgID = int64NullIfZero(r.OrgID)
	data.RepoID = int64NullIfZero(r.RepoID)
	data.Source = types.StringValue(r.Source)
	data.CreatedAt = types.Int64Value(r.CreatedAt)
	data.UpdatedAt = types.Int64Value(r.UpdatedAt)
	data.Events = stringsToList(r.Events)
	data.Images = stringsToList(r.Images)
}
