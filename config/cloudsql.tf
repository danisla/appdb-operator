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
  project = "${var.project == "" ? data.google_client_config.current.project : var.project }"
  name    = "${var.name}-${random_id.name.hex}"
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

output "name" {
  value = "${local.name}"
}

output "connection" {
  value = "${local.project}:${var.region}:${module.db-instance.instance_name}"
}

output "admin_pass" {
  value = "${module.db-instance.generated_user_password}"
}
