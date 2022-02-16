package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSwaggerValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		request    string
		statusCode int
		message    string
	}{
		{
			name: "valid",
			request: `{
				"task": {
					"name": "valid_task",
					"condition": {
						"services": {
							"names": [
								"api"
							]
						}
					},
					"module": "./example-module"
				}
			}`,
			statusCode: http.StatusOK,
			message:    "",
		},
		{
			name:       "empty request",
			request:    "",
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: value is required but missing\n",
		},
		{
			name: "missing required fields",
			request: `{
				"task": {
					"name": "valid_task",
					"module": "./example-module"
				}
			}`,
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: doesn't match the schema: Error at \"/task/condition\": property \"condition\" is missing\n",
		},
		{
			name: "invalid field",
			request: `{
				"task": {
					"name": "valid_task",
					"condition": {
						"services": {
							"names": [
								"api"
							]
						}
					},
					"module": "./example-module",
                    "services": ["api"]
				}
			}`,
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: doesn't match the schema: Error at \"/task\": property \"services\" is unsupported\n",
		},
		{
			name: "invalid condition services field",
			request: `{
				"task": {
					"name": "valid_task",
					"condition": {
						"services": {
                            "source_includes_var": true,
							"names": [
								"api"
							]
						}
					},
					"module": "./example-module"
				}
			}`,
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: doesn't match the schema: Error at \"/task/condition/services\": property \"source_includes_var\" is unsupported\n",
		},
		{
			name: "invalid condition kv field",
			request: `{
				"task": {
					"name": "valid_task",
					"module_input": {
						"services": {
							"names": [
								"api"
							]
						}
					},
					"condition": {
						"consul_kv": {
							"path": "this/path",
                            "source_includes_var": true
						}
					},
					"module": "./example-module"
				}
			}`,
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: doesn't match the schema: Error at \"/task/condition/consul_kv\": property \"source_includes_var\" is unsupported\n",
		},
		{
			name: "invalid condition catalog services field",
			request: `{
				"task": {
					"name": "valid_task",
					"module_input": {
						"services": {
							"names": [
								"api"
							]
						}
					},
					"condition": {
						"catalog_services": {
							"source_includes_var": true,
							"regexp": "*"
						}
					},
					"module": "./example-module"
				}
			}`,
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: doesn't match the schema: Error at \"/task/condition/catalog_services\": property \"source_includes_var\" is unsupported\n",
		},
		{
			name: "invalid module input consul kv field",
			request: `{
				"task": {
					"name": "valid_task",
					"condition": {
						"services": {
							"names": [
								"api"
							]
						}
					},
					"module_input": {
						"consul_kv": {
                            "invalid_field": true,
							"path": "this/path"
						}
					},
					"module": "./example-module"
				}
			}`,
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: doesn't match the schema: Error at \"/task/module_input/consul_kv\": property \"invalid_field\" is unsupported\n",
		},
		{
			name: "invalid module input services field",
			request: `{
				"task": {
					"name": "valid_task",
					"condition": {
						"services": {
							"names": [
								"api"
							]
						}
					},
					"module_input": {
						"services": {
                            "invalid_field": true,
							"names": [
								"api"
							]
						}
					},
					"module": "./example-module"
				}
			}`,
			statusCode: http.StatusBadRequest,
			message:    "request body has an error: doesn't match the schema: Error at \"/task/module_input/services\": property \"invalid_field\" is unsupported\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := runValidationMiddlewarePost(t, tc.statusCode, tc.request, http.MethodPost, "/v1/tasks")
			assert.Contains(t, tc.message, resp.Body.String())
		})
	}
}

func runValidationMiddlewarePost(t *testing.T, expectedStatus int, request, method, path string) *httptest.ResponseRecorder {
	r := strings.NewReader(request)
	req, err := http.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	resp := httptest.NewRecorder()

	// create a handler to use as "next" which does nothing
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handlerToTest := withSwaggerValidate(nextHandler)
	handlerToTest.ServeHTTP(resp, req)

	require.Equal(t, expectedStatus, resp.Code)

	return resp
}
