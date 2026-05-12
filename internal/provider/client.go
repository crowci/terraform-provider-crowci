package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// datasourceWithClient is embedded in data sources to provide the Configure method.
type datasourceWithClient struct {
	client *crowciClient
}

func (d *datasourceWithClient) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// resourceWithClient is embedded in resources to provide the Configure method.
type resourceWithClient struct {
	client *crowciClient
}

func (r *resourceWithClient) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// fetchAllPages retrieves all pages from a paginated API endpoint.
// baseURL must not contain page/perPage query parameters.
func fetchAllPages[T any](ctx context.Context, client *crowciClient, baseURL string) ([]T, error) {
	var all []T
	for pageNum := 1; ; pageNum++ {
		url := fmt.Sprintf("%s?page=%d&perPage=50", baseURL, pageNum)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build request: %w", err)
		}
		httpResp, err := client.HTTPClient.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("API request failed: %w", err)
		}
		if httpResp.StatusCode != http.StatusOK {
			httpResp.Body.Close()
			return nil, fmt.Errorf("unexpected status %d", httpResp.StatusCode)
		}
		var items []T
		err = json.NewDecoder(httpResp.Body).Decode(&items)
		httpResp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		all = append(all, items...)
		if len(items) < 50 {
			break
		}
	}
	return all, nil
}
