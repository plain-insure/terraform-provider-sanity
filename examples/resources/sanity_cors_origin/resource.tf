resource "sanity_cors_origin" "external_app" {
  project           = sanity_project.main.id
  origin            = "https://example.com"
  allow_credentials = true
}
