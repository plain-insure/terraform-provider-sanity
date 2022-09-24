resource "sanity_project_token" "deployer" {
  project   = var.project_id
  label     = "Deployer token"
  role_name = "deploy-studio"
}
