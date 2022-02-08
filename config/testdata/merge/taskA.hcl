terraform_provider "tf_providerA" { }

terraform_provider "tf_providerB" { }

task {
  name = "taskA"
  condition "services" {
    names = ["serviceA", "serviceB"]
  }
}
