package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/posener/complete"
)

func processEOFError(scheme string, err error) error {
	if strings.Contains(err.Error(), "EOF") && scheme == api.HTTPScheme {
		err = fmt.Errorf("%s. Scheme %s was used, "+
			"client may have sent an HTTP request to an HTTPS server. This error can be caused by a client using "+
			"HTTP to connect to a CTS server with TLS enabled, consider using HTTPS scheme instead", err, scheme)
	}

	return err
}

// mergeAutocompleteFlags is used to join multiple flag completion sets.
func mergeAutocompleteFlags(flags ...complete.Flags) complete.Flags {
	merged := make(map[string]complete.Predictor, len(flags))
	for _, f := range flags {
		for k, v := range f {
			merged[k] = v
		}
	}
	return merged
}

func getTasks(ctx context.Context, client oapigen.ClientWithResponsesInterface) (api.TasksResponse, error) {
	resp, err := client.GetAllTasksWithResponse(ctx)
	if err != nil {
		return api.TasksResponse{}, err
	}

	if resp.JSON200 == nil {
		return api.TasksResponse{}, fmt.Errorf("nil response returned with status %s", resp.Status())
	}

	return api.TasksResponse(*resp.JSON200), nil
}
