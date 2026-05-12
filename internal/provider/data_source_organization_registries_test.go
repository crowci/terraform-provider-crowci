package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationRegistriesDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_organization_registry" "registry_a" {
  org_id   = ` + testOrgID + `
  address  = "docker.io"
  username = "acc-test-user-a"
  password = "acc-test-password-a"
}

resource "crowci_organization_registry" "registry_b" {
  org_id   = ` + testOrgID + `
  address  = "ghcr.io"
  username = "acc-test-user-b"
  password = "acc-test-password-b"
  depends_on = [crowci_organization_registry.registry_a]
}

data "crowci_organization_registries" "test" {
  org_id = ` + testOrgID + `
  depends_on = [
    crowci_organization_registry.registry_a,
    crowci_organization_registry.registry_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_organization_registries.test", "registries.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_organization_registries.test", "registries.*", map[string]string{
						"address": "docker.io",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_organization_registries.test", "registries.*", map[string]string{
						"address": "ghcr.io",
					}),
				),
			},
		},
	})
}
