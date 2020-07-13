service {
  name = "serviceC"
  description = "descriptionC"
}

task {
  name = "taskB"
  services = ["serviceC", "serviceD"]
}
