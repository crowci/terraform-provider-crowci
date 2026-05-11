package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserRepositoriesAvailableDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_user_repositories_available" "available" {}

resource "crowci_repository" "test" {
  forge_remote_id = data.crowci_user_repositories_available.available.repositories[0].forge_remote_id
}

data "crowci_user_repositories_available" "test" {
  depends_on = [crowci_repository.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_user_repositories_available.test", "repositories.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_user_repositories_available.test", "repositories.*", map[string]string{
						"active": "true",
					}),
				),
			},
		},
	})
}

func TestAccUserRepositoriesAvailableDataSource_onlyUnenabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_user_repositories_available" "available" {}

resource "crowci_repository" "test" {
  forge_remote_id = data.crowci_user_repositories_available.available.repositories[0].forge_remote_id
}

data "crowci_user_repositories_available" "unenabled" {
  only_unenabled = true
  depends_on     = [crowci_repository.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_user_repositories_available.unenabled", "repositories.#"),
				),
			},
		},
	})
}
