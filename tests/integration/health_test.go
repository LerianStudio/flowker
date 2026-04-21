// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"net/http"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	client := httpClient()
	urls := []string{"/health", "/health/live", "/health/ready"}

	for _, path := range urls {
		path := path // capture range variable
		resp, err := client.Get(baseURL() + path)

		if err != nil {
			t.Fatalf("GET %s failed: %v", path, err)
		}

		statusCode := resp.StatusCode
		resp.Body.Close()

		if statusCode != http.StatusOK {
			t.Fatalf("%s returned %d", path, statusCode)
		}
	}
}
