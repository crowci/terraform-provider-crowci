package provider_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGlobalRegistryResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_registry" "test" {
  address  = "docker.io"
  username = "acc-test-user"
  password = "acc-test-password"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_global_registry.test", "id"),
					resource.TestCheckResourceAttr("crowci_global_registry.test", "address", "docker.io"),
					resource.TestCheckResourceAttr("crowci_global_registry.test", "username", "acc-test-user"),
					resource.TestCheckResourceAttrSet("crowci_global_registry.test", "created_at"),
					resource.TestCheckResourceAttrSet("crowci_global_registry.test", "updated_at"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_global_registry" "test" {
  address  = "docker.io"
  username = "acc-test-user"
  password = "acc-test-password"
}

data "crowci_global_registry" "test" {
  address = crowci_global_registry.test.address
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_global_registry.test", "id",
						"crowci_global_registry.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_global_registry.test", "address",
						"crowci_global_registry.test", "address",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_global_registry.test", "username",
						"crowci_global_registry.test", "username",
					),
				),
			},
		},
	})
}

func TestAccGlobalRegistryResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_registry" "test" {
  address  = "ghcr.io"
  username = "acc-test-user"
  password = "initial-password"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_global_registry.test", "username", "acc-test-user"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_global_registry" "test" {
  address  = "ghcr.io"
  username = "acc-test-user-updated"
  password = "updated-password"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_global_registry.test", "username", "acc-test-user-updated"),
				),
			},
		},
	})
}

func TestAccGlobalRegistryResource_delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkGlobalRegistryDestroyed("crowci_global_registry.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_registry" "test" {
  address  = "quay.io"
  username = "acc-test-user"
  password = "acc-test-password"
}
`,
				Check: resource.TestCheckResourceAttrSet("crowci_global_registry.test", "id"),
			},
		},
	})
}

func TestAccGlobalRegistryResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_registry" "test" {
  address  = "registry.example.com"
  username = "acc-test-user"
  password = "acc-test-password"
}
`,
			},
			{
				ResourceName:            "crowci_global_registry.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           "registry.example.com",
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func checkGlobalRegistryDestroyed(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		address := rs.Primary.Attributes["address"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/registries/%s", os.Getenv("CROWCI_HOST"), address), nil)
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
			return fmt.Errorf("global registry %q still exists, got status %d", address, resp.StatusCode)
		}
		return nil
	}
}
