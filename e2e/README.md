# End-to-End Tests

All files found in the e2e package have a build constraint and requires the build tag `-tags e2e`

The e2e tests require Consul and Terraform installed.

- `consul` found in `$PATH`
- `terraform` in the `./e2e` directory
  - Running the test `TestE2EBasic` first will install Terraform and can be used by other tests.

## Run

Run all e2e tests using the make target found in the [Makefile](../Makefile)

```sh
$ make test-e2e
==> Testing consul-terraform-sync (e2e)
=== RUN   TestE2E_StatusEndpoints
=== PAUSE TestE2E_StatusEndpoints
=== RUN   TestE2E_TaskEndpoints_UpdateEnableDisable
...
```

Run individual e2e tests by running go test and using the `-run` flag with the name of the test or regular expression matching the targeted e2e test names.

```sh
$ go test ./e2e -tags e2e -run TestE2EBasic
```

# Benchmark Tests

Benchmark tests for Consul-Terraform-Sync resides in the project path [`./e2e/benchmarks`](benchmarks). All files found in the benchmark package have a build constraint and requires the build tag `-tags e2e`.

The benchmark tests require Consul and Terraform installed.

- `consul` found in `$PATH`
- `terraform` in the `./e2e` directory

The benchmark tests are run weekly in GitHub Actions from the main branch.

## Run

Run all benchmarks using the make target found in the [Makefile](../Makefile)

```sh
$ make test-benchmarks
```

Run individual benchmarks by running go test and using the `-bench` flag with the name of the benchmark or regular expression matching the targeted benchmark names.

```sh
$ go test ./e2e/benchmarks/ -bench BenchmarkTasksConcurrent* -benchtime=6m -tags e2e
[INFO] freeport: detected ephemeral port range of [49152, 65535]
[INFO] freeport: reducing max blocks from 30 to 26 to avoid the ephemeral port range
goos: darwin
goarch: amd64
pkg: github.com/hashicorp/consul-terraform-sync/e2e/benchmarks
cpu: Intel(R) Core(TM) i7-8559U CPU @ 2.70GHz
BenchmarkTasksConcurrent_t01_s50/task_setup-8         	      10	 112091112 ns/op
BenchmarkTasksConcurrent_t01_s50/once_mode-8          	       1	5113840550 ns/op
...
```
