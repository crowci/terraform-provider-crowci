package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserAccessTokenDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
data "crowci_user_access_token" "test" {
  token_id = 999999
}
`,
				ExpectError: regexp.MustCompile(`Unexpected API response`),
			},
		},
	})
}
