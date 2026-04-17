data "crowci_user_access_token" "example" {
  token_id = 1
}

output "token_name" {
  value = data.crowci_user_access_token.example.name
}

output "token_scopes" {
  value = data.crowci_user_access_token.example.scopes
}
