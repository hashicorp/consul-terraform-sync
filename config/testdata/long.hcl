log_level = "ERR"
inspect_mode = true

syslog {
  enabled = true
  name = "syslog"
}

consul {
  address = "consul-example.com"
  auth {
    enabled = true
    username = "username"
    password = "password"
  }
  kv_path = "kv_path"
  tls {
    ca_cert = "ca_cert"
    ca_path = "ca_path"
    enabled = true
    key = "key"
    server_name = "server_name"
    verify = false
  }
  token = "token"
  transport {
    dial_keep_alive = "5s"
    dial_timeout = "10s"
    disable_keep_alives = false
    idle_conn_timeout = "1m"
    max_idle_conns_per_host = 5
    tls_handshake_timeout = "10s"
  }
}

driver "terraform" {
  log_level = "warn"
  path = "path"
  data_dir = "data"
  working_dir = "working"
  skip_verify = true
  backend "consul" {
    address = "consul-example.com"
    path = "kv-path/terraform"
    gzip = true
  }
}

service {
  name = "serviceA"
  description = "descriptionA"
}

service {
  name = "serviceB"
  namespace = "teamB"
  description = "descriptionB"
}

provider "X" {}

task {
  name = "task"
  description = "automate services for X to do Y"
  services = ["serviceA", "serviceB", "serviceC"]
  providers = ["X"]
  source = "Y"
}
