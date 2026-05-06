resource "crowci_repository" "example" {
  forge_remote_id = "123456"

  timeout    = 120
  visibility = "private"

  trusted = {
    network  = false
    security = false
    volumes  = false
  }
}
