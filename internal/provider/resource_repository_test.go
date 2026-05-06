package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const testForgeRemoteID = "1"

func TestAccRepositoryResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoDestroyed("crowci_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_repository.test", "id"),
					resource.TestCheckResourceAttr("crowci_repository.test", "forge_remote_id", testForgeRemoteID),
					resource.TestCheckResourceAttrSet("crowci_repository.test", "name"),
					resource.TestCheckResourceAttrSet("crowci_repository.test", "full_name"),
					resource.TestCheckResourceAttrSet("crowci_repository.test", "owner"),
					resource.TestCheckResourceAttrSet("crowci_repository.test", "clone_url_ssh"),
					resource.TestCheckResourceAttr("crowci_repository.test", "active", "true"),
					resource.TestCheckResourceAttrSet("crowci_repository.test", "visibility"),
					resource.TestCheckResourceAttrSet("crowci_repository.test", "require_approval"),
				),
			},
		},
	})
}

func TestAccRepositoryResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoDestroyed("crowci_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
  timeout         = 60
}
`,
				Check: resource.TestCheckResourceAttr("crowci_repository.test", "timeout", "60"),
			},
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
  timeout         = 120
  visibility      = "private"
  trusted = {
    network  = false
    security = false
    volumes  = false
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_repository.test", "timeout", "120"),
					resource.TestCheckResourceAttr("crowci_repository.test", "visibility", "private"),
					resource.TestCheckResourceAttr("crowci_repository.test", "trusted.network", "false"),
					resource.TestCheckResourceAttr("crowci_repository.test", "trusted.security", "false"),
					resource.TestCheckResourceAttr("crowci_repository.test", "trusted.volumes", "false"),
				),
			},
		},
	})
}

func TestAccRepositoryResource_delete(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoDestroyed("crowci_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_repository.test", "id"),
					resource.TestCheckResourceAttr("crowci_repository.test", "active", "true"),
				),
			},
			{
				Config: testProviderBlock,
				Check: func(s *terraform.State) error {
					if _, ok := s.RootModule().Resources["crowci_repository.test"]; ok {
						return fmt.Errorf("crowci_repository.test still in state after deletion")
					}
					return nil
				},
			},
		},
	})
}

func TestAccRepositoryResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkRepoDestroyed("crowci_repository.test"),
		Steps: []resource.TestStep{
			{
				Config: testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}
`,
			},
			{
				ResourceName:      "crowci_repository.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources["crowci_repository.test"].Primary.Attributes["id"], nil
				},
			},
		},
	})
}

func checkRepoDestroyed(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		id := rs.Primary.Attributes["id"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/repos/%s", os.Getenv("CROWCI_HOST"), id), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+os.Getenv("CROWCI_TOKEN"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil
		}
		if resp.StatusCode == http.StatusOK {
			var repo struct {
				Active bool `json:"active"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&repo); err == nil && !repo.Active {
				return nil
			}
		}
		return fmt.Errorf("repository %s is still active", id)
	}
}
