data "crowci_forges" "all" {}

output "forge_urls" {
  value = [for f in data.crowci_forges.all.forges : f.url]
}

output "forge_types" {
  value = [for f in data.crowci_forges.all.forges : { id = f.id, type = f.type }]
}
