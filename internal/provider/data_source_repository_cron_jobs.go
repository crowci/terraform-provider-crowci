package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*repositoryCronJobsDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*repositoryCronJobsDataSource)(nil)

func NewRepositoryCronJobsDataSource() datasource.DataSource {
	return &repositoryCronJobsDataSource{}
}

type repositoryCronJobsDataSource struct {
	datasourceWithClient
}

type repositoryCronJobsDataSourceModel struct {
	RepoID   types.Int64              `tfsdk:"repo_id"`
	CronJobs []cronJobItemModel       `tfsdk:"cron_jobs"`
}

type cronJobItemModel struct {
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

func (d *repositoryCronJobsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_cron_jobs"
}

func (d *repositoryCronJobsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches all cron jobs for a specific repository on the Crow CI server.",
		Attributes: map[string]schema.Attribute{
			"repo_id": schema.Int64Attribute{
				Required:    true,
				Description: "ID of the repository to list cron jobs for.",
			},
			"cron_jobs": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: repositoryCronJobSchemaAttrs(),
				},
			},
		},
	}
}

func (d *repositoryCronJobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryCronJobsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	all, err := fetchAllPages[cronJobAPIResponse](ctx, d.client, fmt.Sprintf("%s/api/v1/repos/%d/cron", d.client.Host, data.RepoID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to fetch cron jobs", err.Error())
		return
	}

	cronJobs := make([]cronJobItemModel, len(all))
	for i, c := range all {
		cronJobs[i] = cronJobItemModel{
			ID:        types.Int64Value(c.ID),
			Name:      types.StringValue(c.Name),
			Schedule:  types.StringValue(c.Schedule),
			Branch:    types.StringValue(c.Branch),
			CreatorID: types.Int64Value(c.CreatorID),
			NextExec:  types.Int64Value(c.NextExec),
			Created:   types.Int64Value(c.Created),
			FailCount: types.Int64Value(c.FailCount),
			FailMsg:   types.StringValue(c.FailMsg),
			Disabled:  types.BoolValue(c.Disabled),
		}
	}

	data.CronJobs = cronJobs
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
