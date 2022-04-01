# API Code Generation

## Code Generation Library
[oapi-codegen](https://github.com/deepmap/oapi-codegen) is a library which contains a set of utilities for generating 
Go boilerplate code based on an [OpenAPI 3.0.0](https://github.com/OAI/OpenAPI-Specification/blob/main/versions/3.0.0.md) specification.

## Code Generation
To simplify code generation, the generation is set up using `go:generation` comments in `api/handler.go` and `api/task_lifecycle_client.go`.

To run code generation for CTS, including the API code generation:
1. Install open-api: `go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen`
1. Generate files: `make generate`

## Files
### api/openapi.yaml
Contains the CTS OpenAPI 3.0.0 specification. This file is not generated, and is maintained by the developers of CTS. 
It is used as the blueprint for generating Go API boilerplate

### api/oapigen/server.go
Contains server boilerplate and functions for decoding the openapi spec to support validation middleware. 
This code is generated using the following command: `oapi-codegen  -package oapigen -generate chi-server,spec -o oapigen/server.go openapi.yaml`

### api/oapigen/types.go
Contains the API types which including requests, responses and all other API representations. 
This code is generated using the following command: `
api-codegen  -package oapigen -generate types -o oapigen/types.go openapi.yaml`

### api/oapigen/client.go
Contains client boilerplate.
This code is generated using the following command: `
api-codegen  -package oapigen -generate client -o oapigen/client.go openapi.yaml`
