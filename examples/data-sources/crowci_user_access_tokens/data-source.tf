data "crowci_user_access_tokens" "all" {}

output "token_names" {
  value = [for t in data.crowci_user_access_tokens.all.tokens : t.name]
}

output "token_ids" {
  value = [for t in data.crowci_user_access_tokens.all.tokens : t.id]
}
