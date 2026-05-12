package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	client *crowciClient
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
	cronJobAttrs := map[string]schema.Attribute{
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
					Attributes: cronJobAttrs,
				},
			},
		},
	}
}

func (d *repositoryCronJobsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = client
}

func (d *repositoryCronJobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data repositoryCronJobsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var all []cronJobAPIResponse

	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/api/v1/repos/%d/cron?page=%d&perPage=50", d.client.Host, data.RepoID.ValueInt64(), page)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			resp.Diagnostics.AddError("Failed to build request", err.Error())
			return
		}

		httpResp, err := d.client.HTTPClient.Do(httpReq)
		if err != nil {
			resp.Diagnostics.AddError("API request failed", err.Error())
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			resp.Diagnostics.AddError(
				"Unexpected API response",
				fmt.Sprintf("GET /repos/%d/cron returned status %d", data.RepoID.ValueInt64(), httpResp.StatusCode),
			)
			return
		}

		var pageResults []cronJobAPIResponse
		if err := json.NewDecoder(httpResp.Body).Decode(&pageResults); err != nil {
			resp.Diagnostics.AddError("Failed to decode response", err.Error())
			return
		}

		all = append(all, pageResults...)
		if len(pageResults) < 50 {
			break
		}
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
