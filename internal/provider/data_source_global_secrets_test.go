package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGlobalSecretsDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_global_secret" "secret_a" {
  name   = "acc-test-secrets-list-a"
  value  = "value-a"
  events = ["push"]
}

resource "crowci_global_secret" "secret_b" {
  name   = "acc-test-secrets-list-b"
  value  = "value-b"
  events = ["push", "tag"]
}

data "crowci_global_secrets" "test" {
  depends_on = [
    crowci_global_secret.secret_a,
    crowci_global_secret.secret_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_global_secrets.test", "secrets.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_global_secrets.test", "secrets.*", map[string]string{
						"name": "acc-test-secrets-list-a",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_global_secrets.test", "secrets.*", map[string]string{
						"name": "acc-test-secrets-list-b",
					}),
				),
			},
		},
	})
}