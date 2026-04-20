package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccOrganizationDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_organization" "test" {
  id = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.crowci_organization.test", "id", "1"),
					resource.TestCheckResourceAttrSet("data.crowci_organization.test", "name"),
					resource.TestCheckResourceAttrSet("data.crowci_organization.test", "is_user"),
				),
			},
		},
	})
}

func TestAccOrganizationDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_organization" "test" {
  id = 999999
}
`,
				// The API returns an empty org rather than 404
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.crowci_organization.test", "id", "999999"),
					resource.TestCheckResourceAttr("data.crowci_organization.test", "name", ""),
					resource.TestCheckResourceAttr("data.crowci_organization.test", "is_user", "false"),
				),
			},
		},
	})
}
