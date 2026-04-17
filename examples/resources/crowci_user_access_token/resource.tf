resource "crowci_user_access_token" "ci_bot" {
  name = "ci-bot"
  scopes = [
    "admin:read",
    "admin:write",
    "repo:read",
    "repo:write",
    "repo:admin",
    "user:read",
    "user:write"
  ]
}

# With optional expiry and repo scope
resource "crowci_user_access_token" "repo_scoped" {
  name       = "repo-deploy"
  scopes     = ["repo:admin"]
  repo_id    = 42
  expires_at = 1893456000
}

output "ci_bot_token" {
  value     = crowci_user_access_token.ci_bot.token
  sensitive = true
}
