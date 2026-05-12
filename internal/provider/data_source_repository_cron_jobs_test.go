package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRepositoryCronJobsDataSource_list(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testCronJobConfig("acc-test-cron-list-a", "@daily", "") + `
resource "crowci_repository_cron_job" "test_b" {
  repo_id  = crowci_repository.test.id
  name     = "acc-test-cron-list-b"
  schedule = "@weekly"
  depends_on = [crowci_repository_cron_job.test]
}

data "crowci_repository_cron_jobs" "test" {
  repo_id = crowci_repository.test.id
  depends_on = [
    crowci_repository_cron_job.test,
    crowci_repository_cron_job.test_b,
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.crowci_repository_cron_jobs.test", "cron_jobs.#"),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_repository_cron_jobs.test", "cron_jobs.*", map[string]string{
						"name": "acc-test-cron-list-a",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.crowci_repository_cron_jobs.test", "cron_jobs.*", map[string]string{
						"name": "acc-test-cron-list-b",
					}),
				),
			},
		},
	})
}
