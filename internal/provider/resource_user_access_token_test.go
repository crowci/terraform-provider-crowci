package provider_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccUserAccessTokenResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_user_access_token" "test" {
  name   = "acc-test-token"
  scopes = ["repo:read"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_user_access_token.test", "id"),
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "name", "acc-test-token"),
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "scopes.#", "1"),
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "scopes.0", "repo:read"),
					resource.TestCheckResourceAttrSet("crowci_user_access_token.test", "token"),
					resource.TestCheckResourceAttrSet("crowci_user_access_token.test", "user_id"),
					resource.TestCheckResourceAttrSet("crowci_user_access_token.test", "created_at"),
					resource.TestCheckResourceAttrSet("crowci_user_access_token.test", "updated_at"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_user_access_token" "test" {
  name   = "acc-test-token"
  scopes = ["repo:read"]
}

data "crowci_user_access_token" "test" {
  token_id = crowci_user_access_token.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_user_access_token.test", "id",
						"crowci_user_access_token.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_user_access_token.test", "name",
						"crowci_user_access_token.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_user_access_token.test", "scopes.#",
						"crowci_user_access_token.test", "scopes.#",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_user_access_token.test", "user_id",
						"crowci_user_access_token.test", "user_id",
					),
				),
			},
		},
	})
}

func TestAccUserAccessTokenResource_delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkTokenDestroyed("crowci_user_access_token.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_user_access_token" "test" {
  name   = "acc-test-token-delete"
  scopes = ["repo:read"]
}
`,
				Check: resource.TestCheckResourceAttrSet("crowci_user_access_token.test", "id"),
			},
		},
	})
}

func checkTokenDestroyed(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		id := rs.Primary.Attributes["id"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/user/access-tokens/%s", os.Getenv("CROWCI_HOST"), id), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+os.Getenv("CROWCI_TOKEN"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("token %s still exists, got status %d", id, resp.StatusCode)
		}
		return nil
	}
}

func TestAccUserAccessTokenResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_user_access_token" "test" {
  name   = "acc-test-token"
  scopes = ["repo:read"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "name", "acc-test-token"),
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "scopes.#", "1"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_user_access_token" "test" {
  name   = "acc-test-token-updated"
  scopes = ["repo:read", "repo:write"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "name", "acc-test-token-updated"),
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "scopes.#", "2"),
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "scopes.0", "repo:read"),
					resource.TestCheckResourceAttr("crowci_user_access_token.test", "scopes.1", "repo:write"),
					// token secret must be preserved across updates (UseStateForUnknown)
					resource.TestCheckResourceAttrSet("crowci_user_access_token.test", "token"),
				),
			},
		},
	})
}

func TestAccUserAccessTokensDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy: func(s *terraform.State) error {
			for _, name := range []string{
				"crowci_user_access_token.token_a",
				"crowci_user_access_token.token_b",
			} {
				if err := checkTokenDestroyed(name)(s); err != nil {
					return err
				}
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_user_access_token" "token_a" {
  name   = "acc-test-token-list-a"
  scopes = ["repo:read"]
}

resource "crowci_user_access_token" "token_b" {
  name   = "acc-test-token-list-b"
  scopes = ["repo:read", "repo:write"]
}

data "crowci_user_access_tokens" "test" {
  depends_on = [
    crowci_user_access_token.token_a,
    crowci_user_access_token.token_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_user_access_tokens.test", "tokens.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_user_access_tokens.test", "tokens.*", map[string]string{
						"name": "acc-test-token-list-a",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_user_access_tokens.test", "tokens.*", map[string]string{
						"name": "acc-test-token-list-b",
					}),
				),
			},
		},
	})
}
