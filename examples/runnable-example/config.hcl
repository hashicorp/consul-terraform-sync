log_level = "INFO"

consul {
  address = "localhost:8500"
}

task {
  name = "example-task"
  description = "Writes the service name, id, and IP address to a file"
  source = "./example-module"
  providers = ["local"]
  services = ["web", "api"]
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

# Optional service block for defining configuration specific
# to a service.
service {
  name = "web"
  cts_user_defined_meta = {
    "test_key" = "test_value"
  }
}
