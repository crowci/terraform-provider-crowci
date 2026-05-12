resource "crowci_organization_registry" "example" {
  org_id   = 1
  address  = "docker.io"
  username = "myuser"
  password = "mypassword"
}
