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
