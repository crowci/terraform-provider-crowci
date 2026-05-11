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

# Activate multiple repositories across multiple forges using a nested local map.
# Top-level keys are forge URLs; second-level keys are owners; values are lists of repo names.

data "crowci_forges" "all" {}

locals {
  repos_by_forge = {
    "https://gitea.example.com" = {
      "my-org" = ["repo-one", "repo-two"]
    }
    "https://github.example.com" = {
      "another-org" = ["repo-three"]
    }
  }

  forge_url_to_id = {
    for f in data.crowci_forges.all.forges : f.url => f.id
  }
}

data "crowci_user_repositories_available" "by_forge" {
  for_each = local.forge_url_to_id
  forge_id = each.value
}

locals {
  repos_to_activate = merge([
    for forge_url, repos_by_owner in local.repos_by_forge : merge([
      for owner, names in repos_by_owner : {
        for name in names :
        "${forge_url}/${owner}/${name}" => one([
          for r in data.crowci_user_repositories_available.by_forge[forge_url].repositories : {
            forge_remote_id = r.forge_remote_id
            forge_id        = r.forge_id
          }
          if r.owner == owner && r.name == name
        ])
      }
    ]...)
  ]...)
}

resource "crowci_repository" "repos" {
  for_each        = local.repos_to_activate
  forge_remote_id = each.value.forge_remote_id
  forge_id        = each.value.forge_id
}
