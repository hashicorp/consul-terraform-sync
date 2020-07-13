service {
  name = "serviceA"
  description = "descriptionA"
}

service {
  name = "serviceB"
  namespace = "teamB"
  description = "descriptionB"
}

task {
  name = "taskA"
  services = ["serviceA", "serviceB"]
}
