resource "crowci_global_secret" "deploy_key" {
  name  = "deploy-key"
  value = "ssh-ed25519 AAAA..."
  events = [
    "cron",
    "deployment",
    "manual",
    "pull_request",
    "push",
    "release",
    "tag"
  ]
}

# With image restriction
resource "crowci_global_secret" "docker_password" {
  name   = "docker-password"
  value  = "s3cr3t"
  events = ["push"]
  images = ["plugins/docker"]
}
