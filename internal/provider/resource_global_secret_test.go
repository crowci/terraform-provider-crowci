package provider_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGlobalSecretResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_secret" "test" {
  name   = "acc-test-secret"
  value  = "super-secret-value"
  events = ["push"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_global_secret.test", "id"),
					resource.TestCheckResourceAttr("crowci_global_secret.test", "name", "acc-test-secret"),
					resource.TestCheckResourceAttr("crowci_global_secret.test", "events.#", "1"),
					resource.TestCheckResourceAttr("crowci_global_secret.test", "events.0", "push"),
					resource.TestCheckResourceAttrSet("crowci_global_secret.test", "created_at"),
					resource.TestCheckResourceAttrSet("crowci_global_secret.test", "updated_at"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_global_secret" "test" {
  name   = "acc-test-secret"
  value  = "super-secret-value"
  events = ["push"]
}

data "crowci_global_secret" "test" {
  name = crowci_global_secret.test.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_global_secret.test", "id",
						"crowci_global_secret.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_global_secret.test", "name",
						"crowci_global_secret.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_global_secret.test", "events.#",
						"crowci_global_secret.test", "events.#",
					),
				),
			},
		},
	})
}

func TestAccGlobalSecretResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_secret" "test" {
  name   = "acc-test-secret-update"
  value  = "initial-value"
  events = ["push"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_global_secret.test", "events.#", "1"),
					resource.TestCheckResourceAttr("crowci_global_secret.test", "events.0", "push"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_global_secret" "test" {
  name   = "acc-test-secret-update"
  value  = "updated-value"
  events = ["push", "tag"]
  images = ["alpine"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_global_secret.test", "events.#", "2"),
					resource.TestCheckResourceAttr("crowci_global_secret.test", "events.1", "tag"),
					resource.TestCheckResourceAttr("crowci_global_secret.test", "images.#", "1"),
					resource.TestCheckResourceAttr("crowci_global_secret.test", "images.0", "alpine"),
				),
			},
		},
	})
}

func TestAccGlobalSecretResource_delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkSecretDestroyed("crowci_global_secret.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_secret" "test" {
  name   = "acc-test-secret-delete"
  value  = "some-value"
  events = ["push"]
}
`,
				Check: resource.TestCheckResourceAttrSet("crowci_global_secret.test", "id"),
			},
		},
	})
}

func TestAccGlobalSecretResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_secret" "test" {
  name   = "acc-test-secret-import"
  value  = "import-value"
  events = ["push"]
}
`,
			},
			{
				ResourceName:      "crowci_global_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["crowci_global_secret.test"].Primary.Attributes["name"], nil
				},
				ImportStateVerifyIgnore: []string{"value"},
			},
		},
	})
}

func checkSecretDestroyed(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		name := rs.Primary.Attributes["name"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/secrets/%s", os.Getenv("CROWCI_HOST"), name), nil)
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
			return fmt.Errorf("secret %q still exists, got status %d", name, resp.StatusCode)
		}
		return nil
	}
}