# tflint configuration for the example Terraform snippets under examples/.
# Run locally with: tflint --recursive
config {
  call_module_type = "none"
}

plugin "terraform" {
  enabled = true
  preset  = "recommended"
}

# The examples under examples/ are intentionally minimal documentation
# snippets, not self-contained root modules, so a few whole-module rules
# do not apply and would only produce noise.
rule "terraform_required_providers" {
  enabled = false
}

rule "terraform_required_version" {
  enabled = false
}

rule "terraform_standard_module_structure" {
  enabled = false
}

# Data-source examples intentionally declare a data block without always
# consuming it in an output, so this rule does not fit the snippets.
rule "terraform_unused_declarations" {
  enabled = false
}
