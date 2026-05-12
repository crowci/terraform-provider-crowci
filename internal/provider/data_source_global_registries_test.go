package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGlobalRegistriesDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_registry" "registry_a" {
  address  = "docker.io"
  username = "acc-test-user-a"
  password = "acc-test-password-a"
}

resource "crowci_global_registry" "registry_b" {
  address  = "ghcr.io"
  username = "acc-test-user-b"
  password = "acc-test-password-b"
  depends_on = [crowci_global_registry.registry_a]
}

data "crowci_global_registries" "test" {
  depends_on = [
    crowci_global_registry.registry_a,
    crowci_global_registry.registry_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_global_registries.test", "registries.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_global_registries.test", "registries.*", map[string]string{
						"address": "docker.io",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_global_registries.test", "registries.*", map[string]string{
						"address": "ghcr.io",
					}),
				),
			},
		},
	})
}
