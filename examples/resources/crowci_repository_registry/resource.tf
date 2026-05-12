resource "crowci_repository_registry" "example" {
  repo_id  = 42
  address  = "docker.io"
  username = "myuser"
  password = "mypassword"
}
