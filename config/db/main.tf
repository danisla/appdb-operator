variable "instance" {}

variable "dbname" {}

variable "charset" {
  default = ""
}

variable "collation" {
  default = ""
}

variable "users" {
  // CSV list of users, will generate random password for each.    
}

variable "user_host" {
  default = "%"
}

resource "google_sql_database" "default" {
  name      = "${var.dbname}"
  instance  = "${var.instance}"
  charset   = "${var.charset}"
  collation = "${var.collation}"
}

locals {
  users = ["${split(",", "${var.users}")}"]
}

resource "random_id" "user-passwords" {
  count       = "${length(local.users)}"
  byte_length = 8

  // TODO: Generate new password if instance changes.
  // keepers {
  //   instance = "${var.instance}"
  // }
}

resource "google_sql_user" "users" {
  count    = "${length(local.users)}"
  name     = "${element(local.users, count.index)}"
  instance = "${var.instance}"
  host     = "${var.user_host}"
  password = "${element(random_id.user-passwords.*.hex, count.index)}"
}

output "user_passwords" {
  value     = "${join(",", random_id.user-passwords.*.hex)}"
  sensitive = true
}
