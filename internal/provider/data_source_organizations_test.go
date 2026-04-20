package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationsDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_organizations" "test" {}
`,
				// Check: resource.ComposeAggregateTestCheckFunc(
				// 	resource.TestCheckResourceAttrSet("data.crowci_organizations.test", "organizations.#"),
				// 	resource.TestCheckResourceAttrSet("data.crowci_organizations.test", "organizations.0.id"),
				// 	resource.TestCheckResourceAttrSet("data.crowci_organizations.test", "organizations.0.name"),
				// ),
				Check: resource.TestCheckResourceAttr("data.crowci_organizations.test", "organizations.#", "0"),
			},
		},
	})
}
