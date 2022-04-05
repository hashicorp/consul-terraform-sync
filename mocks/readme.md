# Mock Code Generation

## Code Generation Library
[mockery](https://github.com/vektra/mockery) is a library which contains a set of utilities for generating interface mocks

## Code Generation
To simplify code generation, the generation should be set up to use `go:generation`

To set this up, add the following line to the file containing the interface to be mocked

//go:generate mockery --name=<name_of_interface_to_mock> --filename=<name_of_file_to_create>  --output=<relative_path_to_mocks>/<package_name> --tags=<comma separated list of build tags>

eg. //go:generate mockery --name=Driver --filename=driver.go  --output=../mocks/driver --tags=tag1,tag2

To run code generation for CTS, including the mock code generation:
1. Install open-api: `go install github.com/vektra/mockery/v2@latest`
1. Generate files: `make generate`
