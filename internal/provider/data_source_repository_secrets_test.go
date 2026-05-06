package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepositorySecretsDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_secret" "secret_a" {
  repo_id = crowci_repository.test.id
  name    = "acc-test-repo-secrets-list-a"
  value   = "value-a"
  events  = ["push"]
}

resource "crowci_repository_secret" "secret_b" {
  repo_id    = crowci_repository.test.id
  name       = "acc-test-repo-secrets-list-b"
  value      = "value-b"
  events     = ["push", "tag"]
  depends_on = [crowci_repository_secret.secret_a]
}

data "crowci_repository_secrets" "test" {
  repo_id = crowci_repository.test.id
  depends_on = [
    crowci_repository_secret.secret_a,
    crowci_repository_secret.secret_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_repository_secrets.test", "secrets.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_repository_secrets.test", "secrets.*", map[string]string{
						"name": "acc-test-repo-secrets-list-a",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_repository_secrets.test", "secrets.*", map[string]string{
						"name": "acc-test-repo-secrets-list-b",
					}),
				),
			},
		},
	})
}
