variable "project" {
  default = ""
}

variable "name" {}

variable "database_version" {}

variable "region" {}

variable "tier" {}

variable "disk_size_gb" {}

variable "disk_type" {
  default = "PD_SSD"
}

data "google_client_config" "current" {}

resource "random_id" "name" {
  byte_length = 2
}

locals {
  project  = "${var.project == "" ? data.google_client_config.current.project : var.project }"
  name     = "${var.name}-${random_id.name.hex}"
  port     = "${substr(var.database_version, 0, 5) == "MYSQL" ? "3306" : "5432"}"
  proxy_sa_key   = "${google_service_account_key.cloudsql-proxy.private_key}"
  proxy_sa_email = "${google_service_account.cloudsql-proxy.email}"
}

module "db-instance" {
  source           = "GoogleCloudPlatform/sql-db/google"
  version          = "1.0.1"
  project          = "${local.project}"
  name             = "${local.name}"
  database_version = "${var.database_version}"
  tier             = "${var.tier}"
  user_name        = "admin"
  disk_size        = "${var.disk_size_gb}"
  disk_type        = "${var.disk_type}"
}

resource "google_service_account" "cloudsql-proxy" {
  project      = "${var.project}"
  account_id   = "${local.name}-proxy"
  display_name = "${local.name} Cloud SQL Proxy"
}

resource "google_service_account_key" "cloudsql-proxy" {
  service_account_id = "${google_service_account.cloudsql-proxy.name}"
  public_key_type    = "TYPE_X509_PEM_FILE"
}

resource "google_project_iam_member" "editor" {
  project = "${var.project}"
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${local.proxy_sa_email}"
}

output "name" {
  value = "${local.name}"
}

output "connection" {
  value = "${local.project}:${var.region}:${module.db-instance.instance_name}"
}

output "admin_pass" {
  value     = "${module.db-instance.generated_user_password}"
  sensitive = true
}

output "port" {
  value = "${local.port}"
}

output "proxy_sa_email" {
  value = "${local.proxy_sa_email}"
}

output "proxy_sa_key" {
  value     = "${local.proxy_sa_key}"
  sensitive = true
}
