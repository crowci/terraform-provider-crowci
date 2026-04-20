package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGlobalSecretDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_global_secret" "test" {
  name = "this-secret-does-not-exist-xyz"
}
`,
				ExpectError: regexp.MustCompile("Secret not found"),
			},
		},
	})
}
