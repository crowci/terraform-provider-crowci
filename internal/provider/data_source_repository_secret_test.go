package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepositorySecretDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_secret" "test" {
  repo_id = crowci_repository.test.id
  name    = "acc-test-repo-secret-ds"
  value   = "super-secret-value"
  events  = ["push"]
}

data "crowci_repository_secret" "test" {
  repo_id = crowci_repository_secret.test.repo_id
  name    = crowci_repository_secret.test.name
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_secret.test", "id",
						"crowci_repository_secret.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_secret.test", "name",
						"crowci_repository_secret.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_secret.test", "repo_id",
						"crowci_repository_secret.test", "repo_id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_secret.test", "events.#",
						"crowci_repository_secret.test", "events.#",
					),
				),
			},
		},
	})
}

func TestAccRepositorySecretDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_repository_secret" "test" {
  repo_id = 1
  name    = "this-secret-does-not-exist-xyz"
}
`,
				ExpectError: regexp.MustCompile("Secret not found"),
			},
		},
	})
}
