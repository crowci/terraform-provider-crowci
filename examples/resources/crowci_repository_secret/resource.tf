resource "crowci_repository_secret" "deploy_key" {
  repo_id = 42
  name    = "deploy-key"
  value   = "ssh-ed25519 AAAA..."
  events  = ["push", "tag"]
}

# With image restriction
resource "crowci_repository_secret" "docker_password" {
  repo_id = 42
  name    = "docker-password"
  value   = "s3cr3t"
  events  = ["push"]
  images  = ["plugins/docker"]
}
