package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccForgesDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_forges" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.crowci_forges.test", "forges.0.id", "1"),
					resource.TestCheckResourceAttrSet("data.crowci_forges.test", "forges.0.type"),
					resource.TestCheckResourceAttrSet("data.crowci_forges.test", "forges.0.url"),
				),
			},
		},
	})
}
