terraform_provider "tf_providerC" { }

task {
  name = "taskB"
  condition "services" {
    names = ["serviceC", "serviceD"]
  }
}
