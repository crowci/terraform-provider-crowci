package provider_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testRepoRegistryConfig(address, username, password string) string {
	return testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_registry" "test" {
  repo_id  = crowci_repository.test.id
  address  = "` + address + `"
  username = "` + username + `"
  password = "` + password + `"
}
`
}

func TestAccRepositoryRegistryResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testRepoRegistryConfig("docker.io", "acc-test-user", "acc-test-password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_repository_registry.test", "id"),
					resource.TestCheckResourceAttr("crowci_repository_registry.test", "address", "docker.io"),
					resource.TestCheckResourceAttr("crowci_repository_registry.test", "username", "acc-test-user"),
					resource.TestCheckResourceAttrSet("crowci_repository_registry.test", "created_at"),
					resource.TestCheckResourceAttrSet("crowci_repository_registry.test", "updated_at"),
				),
			},
			{
				Config: testRepoRegistryConfig("docker.io", "acc-test-user", "acc-test-password") + `
data "crowci_repository_registry" "test" {
  repo_id = crowci_repository_registry.test.repo_id
  address = crowci_repository_registry.test.address
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_registry.test", "id",
						"crowci_repository_registry.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_registry.test", "address",
						"crowci_repository_registry.test", "address",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_registry.test", "username",
						"crowci_repository_registry.test", "username",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_registry.test", "repo_id",
						"crowci_repository_registry.test", "repo_id",
					),
				),
			},
		},
	})
}

func TestAccRepositoryRegistryResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testRepoRegistryConfig("ghcr.io", "acc-test-user", "initial-password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_repository_registry.test", "username", "acc-test-user"),
				),
			},
			{
				Config: testRepoRegistryConfig("ghcr.io", "acc-test-user-updated", "updated-password"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_repository_registry.test", "username", "acc-test-user-updated"),
				),
			},
		},
	})
}

func TestAccRepositoryRegistryResource_delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoRegistryDestroyed("crowci_repository_registry.test"),
		Steps: []resource.TestStep{
			{
				Config: testRepoRegistryConfig("quay.io", "acc-test-user", "acc-test-password"),
				Check:  resource.TestCheckResourceAttrSet("crowci_repository_registry.test", "id"),
			},
		},
	})
}

func TestAccRepositoryRegistryResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testRepoRegistryConfig("registry.example.com", "acc-test-user", "acc-test-password"),
			},
			{
				ResourceName:      "crowci_repository_registry.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["crowci_repository_registry.test"].Primary
					return fmt.Sprintf("%s/%s", rs.Attributes["repo_id"], rs.Attributes["address"]), nil
				},
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func checkRepoRegistryDestroyed(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		repoID := rs.Primary.Attributes["repo_id"]
		address := rs.Primary.Attributes["address"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/repos/%s/registries/%s", os.Getenv("CROWCI_HOST"), repoID, address), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+os.Getenv("CROWCI_TOKEN"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("repository registry %q still exists, got status %d", address, resp.StatusCode)
		}
		return nil
	}
}
