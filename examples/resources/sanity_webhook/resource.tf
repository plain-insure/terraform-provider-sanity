resource "sanity_webhook" "example" {
  project_id      = sanity_project.example.id
  type            = "document"
  name            = "Content Updates Webhook"
  dataset         = "production"
  url             = "https://api.example.com/webhooks/sanity"
  http_method     = "POST"
  include_drafts  = false
  
  rule {
    on = ["create", "update"]
    filter = "_type == 'post'"
    projection = "{_id, _type, title}"
  }
  
  headers = {
    "Authorization" = "Bearer ${var.api_token}"
    "Content-Type"  = "application/json"
  }
  
  secret = var.webhook_secret
  is_disabled_by_user = false
}

# Example transaction webhook
resource "sanity_webhook" "transaction_example" {
  project_id          = sanity_project.example.id
  type                = "transaction"
  name                = "Transaction Webhook"
  dataset             = "production"
  url                 = "https://api.example.com/webhooks/sanity-transactions"
  description         = "Webhook for transaction events"
  is_disabled_by_user = false
}

variable "api_token" {
  description = "API token for webhook authentication"
  type        = string
  sensitive   = true
}

variable "webhook_secret" {
  description = "Secret for webhook signature verification"
  type        = string
  sensitive   = true
}