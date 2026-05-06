package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepositoryDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_repository" "test" {
  id = 999999999
}
`,
				ExpectError: regexp.MustCompile("Repository not found"),
			},
		},
	})
}

func TestAccRepositoryDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

data "crowci_repository" "test" {
  id = crowci_repository.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository.test", "id",
						"crowci_repository.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository.test", "name",
						"crowci_repository.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository.test", "full_name",
						"crowci_repository.test", "full_name",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository.test", "active",
						"crowci_repository.test", "active",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository.test", "visibility",
						"crowci_repository.test", "visibility",
					),
				),
			},
		},
	})
}
