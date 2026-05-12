resource "crowci_repository_cron_job" "example" {
  repo_id  = 42
  name     = "nightly-build"
  schedule = "0 2 * * *"
  branch   = "main"
}
