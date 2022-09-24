terraform {
  required_providers {
    sanity = {
      source = "tessellator/sanity"
    }
  }
  required_version = ">= 1.0"
}

provider "sanity" {
  token = "xxx"
}

resource "sanity_project" "my_blog" {
  name        = "My blog"
  color       = "#0000ff"
  studio_host = "my-cool-blog"

  lifecycle {
    prevent_destroy = true
  }
}

resource "sanity_cors_origin" "external" {
  project           = sanity_project.my_blog.id
  origin            = "https://example.com"
  allow_credentials = true
}

resource "sanity_dataset" "production" {
  project  = sanity_project.my_blog.id
  name     = "production"
  acl_mode = "public"

  lifecycle {
    prevent_destroy = true
  }
}

resource "sanity_project_token" "deployer" {
  project   = sanity_project.my_blog.id
  label     = "Deployer token"
  role_name = "deploy-studio"
}
