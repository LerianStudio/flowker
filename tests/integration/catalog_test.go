// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

type executorSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Version  string `json:"version"`
}

type executorDetail struct {
	executorSummary
	Schema string `json:"schema"`
}

type validationResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error"`
}

type triggerSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type triggerDetail struct {
	triggerSummary
	Schema string `json:"schema"`
}

func TestCatalogExecutorsAndTriggers(t *testing.T) {
	client := httpClient()

	// executors list
	resp, err := client.Get(baseURL() + "/v1/catalog/executors")
	if err != nil {
		t.Fatalf("list executors: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("list executors status = %d", resp.StatusCode)
	}

	var execs []executorSummary
	if err := json.NewDecoder(resp.Body).Decode(&execs); err != nil {
		resp.Body.Close()
		t.Fatalf("decode executors: %v", err)
	}

	resp.Body.Close()

	if len(execs) == 0 {
		t.Fatalf("expected at least one executor")
	}

	// executor detail for tracer.validate-transaction
	resp, err = client.Get(baseURL() + "/v1/catalog/executors/tracer.validate-transaction")
	if err != nil {
		t.Fatalf("get executor: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("get executor status = %d", resp.StatusCode)
	}

	var ed executorDetail
	if err := json.NewDecoder(resp.Body).Decode(&ed); err != nil {
		resp.Body.Close()
		t.Fatalf("decode executor detail: %v", err)
	}

	resp.Body.Close()

	if ed.Schema == "" {
		t.Fatalf("expected schema for executor")
	}

	// validate executor config (valid)
	bodyValid := map[string]any{
		"config": map[string]any{
			"requestId":            "550e8400-e29b-41d4-a716-446655440000",
			"transactionType":      "PIX",
			"amount":               "100",
			"currency":             "BRL",
			"transactionTimestamp": "2026-01-01T00:00:00Z",
			"account": map[string]any{
				"accountId": "550e8400-e29b-41d4-a716-446655440001",
				"type":      "checking",
				"status":    "active",
			},
		},
	}

	buf, _ := json.Marshal(bodyValid)
	resp, err = client.Post(baseURL()+"/v1/catalog/executors/tracer.validate-transaction/validate", "application/json", bytes.NewBuffer(buf))

	if err != nil {
		t.Fatalf("validate executor: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("validate executor status = %d", resp.StatusCode)
	}

	var vr validationResponse
	if err := json.NewDecoder(resp.Body).Decode(&vr); err != nil {
		resp.Body.Close()
		t.Fatalf("decode validation response: %v", err)
	}

	resp.Body.Close()

	if !vr.Valid {
		t.Fatalf("expected validation to be true")
	}

	// triggers list
	resp, err = client.Get(baseURL() + "/v1/catalog/triggers")
	if err != nil {
		t.Fatalf("list triggers: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("list triggers status = %d", resp.StatusCode)
	}

	var trigs []triggerSummary
	if err := json.NewDecoder(resp.Body).Decode(&trigs); err != nil {
		resp.Body.Close()
		t.Fatalf("decode triggers: %v", err)
	}

	resp.Body.Close()

	if len(trigs) == 0 {
		t.Fatalf("expected at least one trigger")
	}

	resp, err = client.Get(baseURL() + "/v1/catalog/triggers/webhook")
	if err != nil {
		t.Fatalf("get trigger: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("get trigger status = %d", resp.StatusCode)
	}

	var td triggerDetail
	if err := json.NewDecoder(resp.Body).Decode(&td); err != nil {
		resp.Body.Close()
		t.Fatalf("decode trigger detail: %v", err)
	}

	resp.Body.Close()

	if td.Schema == "" {
		t.Fatalf("expected schema for trigger")
	}
}
