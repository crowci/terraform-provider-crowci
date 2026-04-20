package provider_test

import (
	"os"
	"testing"

	"terraform-provider-crowci/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const testProviderBlock = `
provider "crowci" {}
`

func TestMain(m *testing.M) {
	os.Setenv("CROWCI_HOST", "http://localhost:8000")
	os.Exit(m.Run())
}

func protoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"crowci": providerserver.NewProtocol6WithError(provider.New()()),
	}
}
