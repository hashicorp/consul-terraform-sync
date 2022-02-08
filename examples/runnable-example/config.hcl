log_level = "INFO"

consul {
  address = "localhost:8500"
}

task {
  name = "example-task"
  description = "Writes the service name, id, and IP address to a file"
  module = "./example-module"
  providers = ["local"]
  condition "services" {
    names = ["web", "api"]
    cts_user_defined_meta = {
      "api" = "api_meta"
      "web" = "web_meta"
    }
  }
  variable_files = []
}

driver "terraform" {
  required_providers {
    local = {
      source = "hashicorp/local"
      version = "2.1.0"
    }
  }
}

terraform_provider "local" {
}
