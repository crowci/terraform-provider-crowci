# All repositories visible across all linked forges
data "crowci_user_repositories_available" "all" {}

# Only repositories not yet enabled in Crow CI
data "crowci_user_repositories_available" "unenabled" {
  only_unenabled = true
}

# Scoped to a specific forge
data "crowci_user_repositories_available" "forge" {
  forge_id = 1
}
