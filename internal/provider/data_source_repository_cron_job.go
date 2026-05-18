package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*repositoryCronJobDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositoryCronJobDataSource)(nil)

func NewRepositoryCronJobDataSource() datasource.DataSource {
	return &repositoryCronJobDataSource{}
}

type repositoryCronJobDataSource struct {
	datasourceWithClient
}

type repositoryCronJobDataSourceModel struct {
	RepoID    types.Int64  `tfsdk:"repo_id"`
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Schedule  types.String `tfsdk:"schedule"`
	Branch    types.String `tfsdk:"branch"`
	CreatorID types.Int64  `tfsdk:"creator_id"`
	NextExec  types.Int64  `tfsdk:"next_exec"`
	Created   types.Int64  `tfsdk:"created"`
	FailCount types.Int64  `tfsdk:"fail_count"`
	FailMsg   types.String `tfsdk:"fail_msg"`
	Disabled  types.Bool   `tfsdk:"disabled"`
}

func repositoryCronJobSchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Cron job ID.",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Cron job name.",
		},
		"schedule": schema.StringAttribute{
			Computed:    true,
			Description: "Cron schedule expression.",
		},
		"branch": schema.StringAttribute{
			Computed:    true,
			Description: "Branch the cron job runs on.",
		},
		"creator_id": schema.Int64Attribute{
			Computed:    true,
			Description: "ID of the user who created this cron job.",
		},
		"next_exec": schema.Int64Attribute{
			Computed:    true,
			Description: "Next scheduled execution time as a Unix timestamp.",
		},
		"created": schema.Int64Attribute{
			Computed:    true,
			Description: "Creation time as a Unix timestamp.",
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
	}
}

func (d *repositoryCronJobDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_cron_job"
}

func (d *repositoryCronJobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := repositoryCronJobSchemaAttrs()
	attrs["repo_id"] = schema.Int64Attribute{
		Required:    true,
		Description: "ID of the repository this cron job belongs to.",
	}
	attrs["id"] = schema.Int64Attribute{
		Required:    true,
		Description: "Cron job ID.",
	}
	resp.Schema = schema.Schema{
		Description: "Get a cron job for a repository by repo ID and cron job ID.",
		Attributes:  attrs,
	}
}

func (d *repositoryCronJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryCronJobDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%d/cron/%d", d.client.Host, data.RepoID.ValueInt64(), data.ID.ValueInt64())
	httpResp, ok := doRequest(ctx, d.client, http.MethodGet, endpoint, nil, []int{http.StatusOK, http.StatusNotFound}, &resp.Diagnostics)
	if !ok { return }
	defer httpResp.Body.Close()

	if httpResp.StatusCode == http.StatusNotFound {
		resp.Diagnostics.AddError(
			"Cron job not found",
			fmt.Sprintf("No cron job with ID %d exists for repository %d.", data.ID.ValueInt64(), data.RepoID.ValueInt64()),
		)
		return
	}

	var result cronJobAPIResponse
	if !decodeJSON(httpResp.Body, &result, &resp.Diagnostics) { return }

	data.Name = types.StringValue(result.Name)
	data.Schedule = types.StringValue(result.Schedule)
	data.Branch = types.StringValue(result.Branch)
	data.CreatorID = types.Int64Value(result.CreatorID)
	data.NextExec = types.Int64Value(result.NextExec)
	data.Created = types.Int64Value(result.Created)
	data.FailCount = types.Int64Value(result.FailCount)
	data.FailMsg = types.StringValue(result.FailMsg)
	data.Disabled = types.BoolValue(result.Disabled)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
