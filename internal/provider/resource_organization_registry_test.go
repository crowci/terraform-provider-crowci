package provider_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccOrganizationRegistryResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_organization_registry" "test" {
  org_id   = ` + testOrgID + `
  address  = "docker.io"
  username = "acc-test-user"
  password = "acc-test-password"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_organization_registry.test", "id"),
					resource.TestCheckResourceAttr("crowci_organization_registry.test", "org_id", testOrgID),
					resource.TestCheckResourceAttr("crowci_organization_registry.test", "address", "docker.io"),
					resource.TestCheckResourceAttr("crowci_organization_registry.test", "username", "acc-test-user"),
					resource.TestCheckResourceAttrSet("crowci_organization_registry.test", "created_at"),
					resource.TestCheckResourceAttrSet("crowci_organization_registry.test", "updated_at"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_organization_registry" "test" {
  org_id   = ` + testOrgID + `
  address  = "docker.io"
  username = "acc-test-user"
  password = "acc-test-password"
}

data "crowci_organization_registry" "test" {
  org_id  = crowci_organization_registry.test.org_id
  address = crowci_organization_registry.test.address
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_organization_registry.test", "id",
						"crowci_organization_registry.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_organization_registry.test", "address",
						"crowci_organization_registry.test", "address",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_organization_registry.test", "username",
						"crowci_organization_registry.test", "username",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_organization_registry.test", "org_id",
						"crowci_organization_registry.test", "org_id",
					),
				),
			},
		},
	})
}

func TestAccOrganizationRegistryResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_organization_registry" "test" {
  org_id   = ` + testOrgID + `
  address  = "ghcr.io"
  username = "acc-test-user"
  password = "initial-password"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_organization_registry.test", "username", "acc-test-user"),
				),
			},
			{
				Config: testProviderBlock + `
resource "crowci_organization_registry" "test" {
  org_id   = ` + testOrgID + `
  address  = "ghcr.io"
  username = "acc-test-user-updated"
  password = "updated-password"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_organization_registry.test", "username", "acc-test-user-updated"),
				),
			},
		},
	})
}

func TestAccOrganizationRegistryResource_delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkOrgRegistryDestroyed(testOrgID, "crowci_organization_registry.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_organization_registry" "test" {
  org_id   = ` + testOrgID + `
  address  = "quay.io"
  username = "acc-test-user"
  password = "acc-test-password"
}
`,
				Check: resource.TestCheckResourceAttrSet("crowci_organization_registry.test", "id"),
			},
		},
	})
}

func TestAccOrganizationRegistryResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_organization_registry" "test" {
  org_id   = ` + testOrgID + `
  address  = "registry.example.com"
  username = "acc-test-user"
  password = "acc-test-password"
}
`,
			},
			{
				ResourceName:      "crowci_organization_registry.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["crowci_organization_registry.test"].Primary
					return fmt.Sprintf("%s/%s", rs.Attributes["org_id"], rs.Attributes["address"]), nil
				},
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func checkOrgRegistryDestroyed(orgID, resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		address := rs.Primary.Attributes["address"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/orgs/%s/registries/%s", os.Getenv("CROWCI_HOST"), orgID, address), nil)
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
			return fmt.Errorf("org registry %q still exists, got status %d", address, resp.StatusCode)
		}
		return nil
	}
}
