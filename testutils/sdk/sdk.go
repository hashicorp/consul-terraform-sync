package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/pkg/errors"
)

// Copy and pasted from consul/sdk/testutil/.
// Only change is switching from *testing.T to testing.TB so it can
// be used in benchmarks.

// AddCheck adds a check to the Consul instance. If the serviceID is
// left empty (""), then the check will be associated with the node.
// The check status may be "passing", "warning", or "critical".
func AddCheck(s *testutil.TestServer, t testing.TB, name, serviceID, status string) {
	chk := &testutil.TestCheck{
		ID:   name,
		Name: name,
		TTL:  "10m",
	}
	if serviceID != "" {
		chk.ServiceID = serviceID
	}

	payload, err := encodePayload(chk)
	if err != nil {
		t.Fatal(err)
	}
	put(s, t, "/v1/agent/check/register", payload)

	UpdateCheck(s, t, name, serviceID, status)
}

func UpdateCheck(s *testutil.TestServer, t testing.TB, name, serviceID, status string) {
	switch status {
	case testutil.HealthPassing:
		put(s, t, "/v1/agent/check/pass/"+name, nil)
	case testutil.HealthWarning:
		put(s, t, "/v1/agent/check/warn/"+name, nil)
	case testutil.HealthCritical:
		put(s, t, "/v1/agent/check/fail/"+name, nil)
	default:
		t.Fatalf("Unrecognized status: %s", status)
	}
}

// encodePayload returns a new io.Reader wrapping the encoded contents
// of the payload, suitable for passing directly to a new request.
func encodePayload(payload interface{}) (io.Reader, error) {
	var encoded bytes.Buffer
	enc := json.NewEncoder(&encoded)
	if err := enc.Encode(payload); err != nil {
		return nil, errors.Wrap(err, "failed to encode payload")
	}
	return &encoded, nil
}

// put performs a new HTTP PUT request.
func put(s *testutil.TestServer, t testing.TB, path string, body io.Reader) *http.Response {
	req, err := http.NewRequest("PUT", url(s, path), body)
	if err != nil {
		t.Fatalf("failed to create PUT request: %s", err)
	}
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		t.Fatalf("failed to make PUT request: %s", err)
	}
	if err := requireOK(resp); err != nil {
		defer resp.Body.Close()
		t.Fatalf("not OK PUT: %s", err)
	}
	return resp
}

// url is a helper function which takes a relative URL and
// makes it into a proper URL against the local Consul server.
func url(s *testutil.TestServer, path string) string {
	if s == nil {
		log.Fatal("s is nil")
	}
	if s.Config == nil {
		log.Fatal("s.Config is nil")
	}
	if s.Config.Ports == nil {
		log.Fatal("s.Config.Ports is nil")
	}
	if s.Config.Ports.HTTP == 0 {
		log.Fatal("s.Config.Ports.HTTP is 0")
	}
	if path == "" {
		log.Fatal("path is empty")
	}
	return fmt.Sprintf("http://127.0.0.1:%d%s", s.Config.Ports.HTTP, path)
}

// requireOK checks the HTTP response code and ensures it is acceptable.
func requireOK(resp *http.Response) error {
	if resp.StatusCode != 200 {
		return fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}
	return nil
}
