package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationSecretsDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_organization_secret" "secret_a" {
  org_id = ` + testOrgID + `
  name   = "acc-test-org-secrets-list-a"
  value  = "value-a"
  events = ["push"]
}

resource "crowci_organization_secret" "secret_b" {
  org_id = ` + testOrgID + `
  name   = "acc-test-org-secrets-list-b"
  value  = "value-b"
  events = ["push", "tag"]
  depends_on = [crowci_organization_secret.secret_a]
}

data "crowci_organization_secrets" "test" {
  org_id = ` + testOrgID + `
  depends_on = [
    crowci_organization_secret.secret_a,
    crowci_organization_secret.secret_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_organization_secrets.test", "secrets.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_organization_secrets.test", "secrets.*", map[string]string{
						"name": "acc-test-org-secrets-list-a",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_organization_secrets.test", "secrets.*", map[string]string{
						"name": "acc-test-org-secrets-list-b",
					}),
				),
			},
		},
	})
}
