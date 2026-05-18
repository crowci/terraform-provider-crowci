package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func stringsToList(ss []string) types.List {
	if ss == nil {
		ss = []string{}
	}
	elems := make([]attr.Value, len(ss))
	for i, s := range ss {
		elems[i] = types.StringValue(s)
	}
	list, _ := types.ListValue(types.StringType, elems)
	return list
}

func int64NullIfZero(v int64) types.Int64 {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(v)
}

// doRequest builds and executes an HTTP request. It sets Content-Type: application/json when body
// is non-nil. If the response status is not in allowedStatuses, it adds a diagnostic error and
// returns (nil, false). Callers must close the returned response body.
func doRequest(
	ctx context.Context,
	client *crowciClient,
	method, endpoint string,
	body []byte,
	allowedStatuses []int,
	diags *diag.Diagnostics,
) (*http.Response, bool) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		diags.AddError("Failed to build request", err.Error())
		return nil, false
	}
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpResp, err := client.HTTPClient.Do(httpReq)
	if err != nil {
		diags.AddError("API request failed", err.Error())
		return nil, false
	}
	for _, s := range allowedStatuses {
		if httpResp.StatusCode == s {
			return httpResp, true
		}
	}
	b, _ := io.ReadAll(httpResp.Body)
	httpResp.Body.Close()
	diags.AddError(
		"Unexpected API response",
		fmt.Sprintf("%s %s returned status %d: %s", method, endpoint, httpResp.StatusCode, b),
	)
	return nil, false
}

// decodeJSON decodes JSON from r into dest. Adds a diagnostic error and returns false on failure.
func decodeJSON[T any](r io.Reader, dest *T, diags *diag.Diagnostics) bool {
	if err := json.NewDecoder(r).Decode(dest); err != nil {
		diags.AddError("Failed to decode response", err.Error())
		return false
	}
	return true
}

// marshalJSON encodes src to JSON. Returns nil and adds a diagnostic error on failure.
func marshalJSON(src any, diags *diag.Diagnostics) []byte {
	b, err := json.Marshal(src)
	if err != nil {
		diags.AddError("Failed to encode request", err.Error())
		return nil
	}
	return b
}

func listToStrings(list types.List) []string {
	elems := list.Elements()
	out := make([]string, len(elems))
	for i, e := range elems {
		out[i] = e.(types.String).ValueString()
	}
	return out
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
