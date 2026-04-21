resource "crowci_organization_secret" "deploy_key" {
  org_id = 1
  name   = "deploy-key"
  value  = "ssh-ed25519 AAAA..."
  events = ["push", "tag"]
}

# With image restriction
resource "crowci_organization_secret" "docker_password" {
  org_id = 1
  name   = "docker-password"
  value  = "s3cr3t"
  events = ["push"]
  images = ["plugins/docker"]
}
