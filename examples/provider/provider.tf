terraform {
  required_providers {
    crowci = {
      source = "registry.terraform.io/crowci/crowci"
    }
  }
}

provider "crowci" {
  host  = "https://ci.example.com"
  token = var.crowci_token
}

variable "crowci_token" {
  type      = string
  sensitive = true
}
