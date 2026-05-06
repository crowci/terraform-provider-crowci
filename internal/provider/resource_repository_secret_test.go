package provider_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRepositorySecretResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoSecretDestroyed(testForgeRemoteID, "crowci_repository_secret.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_secret" "test" {
  repo_id = crowci_repository.test.id
  name    = "acc-test-repo-secret"
  value   = "super-secret-value"
  events  = ["push"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_repository_secret.test", "id"),
					resource.TestCheckResourceAttrSet("crowci_repository_secret.test", "repo_id"),
					resource.TestCheckResourceAttr("crowci_repository_secret.test", "name", "acc-test-repo-secret"),
					resource.TestCheckResourceAttr("crowci_repository_secret.test", "events.#", "1"),
					resource.TestCheckResourceAttr("crowci_repository_secret.test", "events.0", "push"),
					resource.TestCheckResourceAttrSet("crowci_repository_secret.test", "created_at"),
					resource.TestCheckResourceAttrSet("crowci_repository_secret.test", "updated_at"),
				),
			},
		},
	})
}

func TestAccRepositorySecretResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoSecretDestroyed(testForgeRemoteID, "crowci_repository_secret.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_secret" "test" {
  repo_id = crowci_repository.test.id
  name    = "acc-test-repo-secret-update"
  value   = "initial-value"
  events  = ["push"]
}
`,
				Check: resource.TestCheckResourceAttr("crowci_repository_secret.test", "events.#", "1"),
			},
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_secret" "test" {
  repo_id = crowci_repository.test.id
  name    = "acc-test-repo-secret-update"
  value   = "updated-value"
  events  = ["push", "tag"]
  images  = ["alpine"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_repository_secret.test", "events.#", "2"),
					resource.TestCheckResourceAttr("crowci_repository_secret.test", "events.1", "tag"),
					resource.TestCheckResourceAttr("crowci_repository_secret.test", "images.#", "1"),
					resource.TestCheckResourceAttr("crowci_repository_secret.test", "images.0", "alpine"),
					resource.TestCheckResourceAttrSet("crowci_repository_secret.test", "value"),
				),
			},
		},
	})
}

func TestAccRepositorySecretResource_delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoSecretDestroyed(testForgeRemoteID, "crowci_repository_secret.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_secret" "test" {
  repo_id = crowci_repository.test.id
  name    = "acc-test-repo-secret-delete"
  value   = "some-value"
  events  = ["push"]
}
`,
				Check: resource.TestCheckResourceAttrSet("crowci_repository_secret.test", "id"),
			},
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}
`,
				Check: func(s *terraform.State) error {
					if _, ok := s.RootModule().Resources["crowci_repository_secret.test"]; ok {
						return fmt.Errorf("crowci_repository_secret.test still in state after deletion")
					}
					return nil
				},
			},
		},
	})
}

func TestAccRepositorySecretResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_secret" "test" {
  repo_id = crowci_repository.test.id
  name    = "acc-test-repo-secret-import"
  value   = "import-value"
  events  = ["push"]
}
`,
			},
			{
				ResourceName:      "crowci_repository_secret.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["crowci_repository_secret.test"].Primary
					return fmt.Sprintf("%s/%s", rs.Attributes["repo_id"], rs.Attributes["name"]), nil
				},
				ImportStateVerifyIgnore: []string{"value"},
			},
		},
	})
}

func checkRepoSecretDestroyed(repoForgeID, resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		repoID := rs.Primary.Attributes["repo_id"]
		name := rs.Primary.Attributes["name"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/repos/%s/secrets/%s", os.Getenv("CROWCI_HOST"), repoID, name), nil)
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
			return fmt.Errorf("repository secret %q still exists, got status %d", name, resp.StatusCode)
		}
		return nil
	}
}
