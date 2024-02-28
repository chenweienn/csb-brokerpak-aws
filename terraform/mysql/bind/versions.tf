terraform {
  required_providers {
    csbmysql = {
      source  = "registry.terraform.io/cloud-service-broker/csbmysql"
      version = ">= 1.0.0"
    }
    random = {
      source  = "registry.terraform.io/hashicorp/random"
      version = ">= 3.3.2"
    }
  }
}