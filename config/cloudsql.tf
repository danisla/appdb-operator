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

locals {
  project = "${var.project == "" ? data.google_client_config.current.project : var.project }"
}

module "db-instance" {
  source           = "GoogleCloudPlatform/sql-db/google"
  version          = "1.0.1"
  project          = "${local.project}"
  name             = "${var.name}"
  database_version = "${var.database_version}"
  tier             = "${var.tier}"
  user_name        = "admin"
  disk_size        = "${var.disk_size_gb}"
  disk_type        = "${var.disk_type}"
}

output "connection" {
  value = "${local.project}:${var.region}:${module.db-instance.instance_name}"
}

output "admin_pass" {
  value = "${module.db-instance.generated_user_password}"
}
