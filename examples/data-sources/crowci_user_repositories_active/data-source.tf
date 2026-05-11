data "crowci_user_repositories_active" "all" {}

# Access the first active repository
output "first_repo_full_name" {
  value = data.crowci_user_repositories_active.all.repositories[0].full_name
}
