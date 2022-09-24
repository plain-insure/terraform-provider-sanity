resource "sanity_dataset" "production" {
  project  = var.project_id
  name     = "production"
  acl_mode = "public"
}
