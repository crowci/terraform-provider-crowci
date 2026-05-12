package provider_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRepositoryCronJobResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkCronJobDestroyed("crowci_repository_cron_job.test"),
		Steps: []resource.TestStep{
			{
				Config: testCronJobConfig("acc-test-cron", "@daily", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("crowci_repository_cron_job.test", "id"),
					resource.TestCheckResourceAttr("crowci_repository_cron_job.test", "name", "acc-test-cron"),
					resource.TestCheckResourceAttr("crowci_repository_cron_job.test", "schedule", "@daily"),
					resource.TestCheckResourceAttrSet("crowci_repository_cron_job.test", "creator_id"),
					resource.TestCheckResourceAttrSet("crowci_repository_cron_job.test", "created"),
					resource.TestCheckResourceAttrSet("crowci_repository_cron_job.test", "next_exec"),
				),
			},
			{
				Config: testCronJobConfig("acc-test-cron", "@daily", "") + `
data "crowci_repository_cron_job" "test" {
  repo_id = crowci_repository_cron_job.test.repo_id
  id      = crowci_repository_cron_job.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_cron_job.test", "id",
						"crowci_repository_cron_job.test", "id",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_cron_job.test", "name",
						"crowci_repository_cron_job.test", "name",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_cron_job.test", "schedule",
						"crowci_repository_cron_job.test", "schedule",
					),
					resource.TestCheckResourceAttrPair(
						"data.crowci_repository_cron_job.test", "branch",
						"crowci_repository_cron_job.test", "branch",
					),
				),
			},
		},
	})
}

func TestAccRepositoryCronJobResource_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkCronJobDestroyed("crowci_repository_cron_job.test"),
		Steps: []resource.TestStep{
			{
				Config: testCronJobConfig("acc-test-cron-update", "@daily", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_repository_cron_job.test", "schedule", "@daily"),
				),
			},
			{
				Config: testCronJobConfig("acc-test-cron-update", "@weekly", "main"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("crowci_repository_cron_job.test", "schedule", "@weekly"),
					resource.TestCheckResourceAttr("crowci_repository_cron_job.test", "branch", "main"),
				),
			},
		},
	})
}

func TestAccRepositoryCronJobResource_import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		CheckDestroy:             checkCronJobDestroyed("crowci_repository_cron_job.test"),
		Steps: []resource.TestStep{
			{
				Config: testCronJobConfig("acc-test-cron-import", "@daily", ""),
			},
			{
				ResourceName:      "crowci_repository_cron_job.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources["crowci_repository_cron_job.test"].Primary
					return fmt.Sprintf("%s/%s", rs.Attributes["repo_id"], rs.Attributes["id"]), nil
				},
			},
		},
	})
}

func testCronJobConfig(name, schedule, branch string) string {
	branchLine := ""
	if branch != "" {
		branchLine = fmt.Sprintf(`  branch   = %q`, branch)
	}
	return testProviderBlock + `
resource "crowci_repository" "test" {
  forge_remote_id = "` + testForgeRemoteID + `"
}

resource "crowci_repository_cron_job" "test" {
  repo_id  = crowci_repository.test.id
  name     = "` + name + `"
  schedule = "` + schedule + `"
` + branchLine + `
}
`
}

func checkCronJobDestroyed(resourceName string) func(*terraform.State) error {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}

		repoID := rs.Primary.Attributes["repo_id"]
		id := rs.Primary.Attributes["id"]
		req, err := http.NewRequest(http.MethodGet,
			fmt.Sprintf("%s/api/v1/repos/%s/cron/%s", os.Getenv("CROWCI_HOST"), repoID, id), nil)
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
			return fmt.Errorf("cron job %s/%s still exists, got status %d", repoID, id, resp.StatusCode)
		}
		return nil
	}
}
