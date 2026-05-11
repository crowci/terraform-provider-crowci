package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserRepositoriesActiveDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

data "crowci_user_repositories_active" "test" {
  depends_on = [crowci_repository.test]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_user_repositories_active.test", "repositories.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_user_repositories_active.test", "repositories.*", map[string]string{
						"forge_remote_id": testForgeRemoteID,
						"active":          "true",
					}),
				),
			},
		},
	})
}
