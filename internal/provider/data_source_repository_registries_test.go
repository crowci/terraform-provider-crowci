package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepositoryRegistriesDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_registry" "registry_a" {
  repo_id  = crowci_repository.test.id
  address  = "docker.io"
  username = "acc-test-user-a"
  password = "acc-test-password-a"
}

resource "crowci_repository_registry" "registry_b" {
  repo_id  = crowci_repository.test.id
  address  = "ghcr.io"
  username = "acc-test-user-b"
  password = "acc-test-password-b"
  depends_on = [crowci_repository_registry.registry_a]
}

data "crowci_repository_registries" "test" {
  repo_id = crowci_repository.test.id
  depends_on = [
    crowci_repository_registry.registry_a,
    crowci_repository_registry.registry_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_repository_registries.test", "registries.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_repository_registries.test", "registries.*", map[string]string{
						"address": "docker.io",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_repository_registries.test", "registries.*", map[string]string{
						"address": "ghcr.io",
					}),
				),
			},
		},
	})
}
