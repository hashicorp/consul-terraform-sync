# Consul-Terraform-Sync

Consul-Terraform-Sync (CTS) extends Consul to automating your network infrastructure with Terraform. CTS monitors changes to the L7 network layer and uses Terraform to dynamically update your infrastructure. You can customize automation to suit your team's infrastructure needs by defining the resources to change with Terraform modules.

* Documentation: [consul.io/docs/nia](https://www.consul.io/docs/nia)
* GitHub: [github.com/hashicorp/consul-terraform-sync](https://github.com/hashicorp/consul-terraform-sync)
* Community portal: [discuss.hashicorp.com](https://discuss.hashicorp.com/tags/c/consul/29/consul-terraform-sync)

## How to use this image

The Docker image [hashicorp/consul-terraform-sync](https://hub.docker.com/r/hashicorp/consul-terraform-sync) is available to run Consul-Terraform-Sync in a scheduled environment.

Run requirements
* A running Consul agent to for Consul-Terraform-Sync to connect with
* [Configuration](https://www.consul.io/docs/nia/configuration) file for Consul-Terraform-Sync

### Running Consul-Terraform-Sync in Daemon Mode

Start a Consul-Terraform-Sync instance

```
$ docker run --name cts -d --rm \
  -v $(pwd):/consul-terraform-sync/config \
  hashicorp/consul-terraform-sync
```

#### Required
Configuration file(s) set using the Docker `-v` flag:

The `-v` flag binds the current working directory as a volume to the expected path for the container to load configuration files from, `/consul-terraform-sync/config`. If your configration files are in another location, replace `$(pwd)` with the absolute path.

#### Terraform State Files

Upon restarting the container, the snapshot of CTS tasks are deferred to the corresponding Terraform state files. The network driver and the configured [Terraform backend](https://www.consul.io/docs/nia/configuration#backend) determine the location of the state file storage, like Consul KV or Terraform Cloud.

Do not use the local backend when deploying Consul-Terraform-Sync as a Docker container for production. Once the container stops, Terraform state files will not persist.

### Executing the Consul-Terraform-Sync CLI

Consul-Terraform-Sync has a [client CLI](https://www.consul.io/docs/nia/cli/task) that can be used with `docker exec`.

Below is an example running the CLI to enable a task using Docker to a container named `cts`.

```
$ docker exec -t cts consul-terraform-sync task enable task_a
==> Inspecting changes to resource if enabling 'task_a'...

    Generating plan that Consul-Terraform-Sync will use Terraform to execute

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.

==> 'task_a' enable complete!
```

### Environment Variables

The Docker image supports all environment variables named in the [configuration documentation](https://www.consul.io/docs/nia/configuration) and can be passed to the container using the `-e` Docker flag.

```
$ docker run --name cts -d --rm \
  -v $(pwd):/consul-terraform-sync/config \
  -e "CONSUL_HTTP_ADDR=consul.example.com" \
  hashicorp/consul-terraform-sync
```

## How to build this image

The [Dockerfile](https://github.com/hashicorp/consul-terraform-sync/blob/main/docker/Dockerfile) can be used to build a local Docker image with the below command and required argument. `VERSION` is the Consul-Terraform-Sync version to install from [releases.hashicorp.com](https://releases.hashicorp.com/consul-terraform-sync/) and build the image with.

```
$ cd consul-terraform-sync/docker
$ ls
Dockerfile    README.md   docker-entry.point.sh*
$ docker build \
  --build-arg VERSION=0.2.1 \
  --tag local-cts-image .
```
