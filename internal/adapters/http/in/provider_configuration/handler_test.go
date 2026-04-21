// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package providerconfiguration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/api"
	httpProviderConfig "github.com/LerianStudio/flowker/internal/adapters/http/in/provider_configuration"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestApp(t *testing.T, ctrl *gomock.Controller) (*fiber.App, *MockCommandService, *MockQueryService) {
	t.Helper()

	mockCmdSvc := NewMockCommandService(ctrl)
	mockQuerySvc := NewMockQueryService(ctrl)

	app := fiber.New()
	handler, err := httpProviderConfig.NewHandler(mockCmdSvc, mockQuerySvc)
	require.NoError(t, err)

	handler.RegisterRoutes(app.Group("/v1"))

	return app, mockCmdSvc, mockQuerySvc
}

func TestList_InvalidLimit(t *testing.T) {
	tests := []struct {
		name       string
		limitParam string
	}{
		{name: "non-numeric limit", limitParam: "abc"},
		{name: "zero limit", limitParam: "0"},
		{name: "negative limit", limitParam: "-5"},
		{name: "limit too large", limitParam: "101"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			app, _, _ := createTestApp(t, ctrl)

			req := httptest.NewRequest(http.MethodGet, "/v1/provider-configurations?limit="+tt.limitParam, nil)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

			var errResp api.ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errResp)
			require.NoError(t, err)
			assert.Equal(t, constant.ErrInvalidQueryParameter.Error(), errResp.Code)
			assert.Equal(t, "Bad Request", errResp.Title)
		})
	}
}

func TestList_InvalidStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	app, _, _ := createTestApp(t, ctrl)

	req := httptest.NewRequest(http.MethodGet, "/v1/provider-configurations?status=banana", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var errResp api.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, constant.ErrInvalidQueryParameter.Error(), errResp.Code)
	assert.Equal(t, "Bad Request", errResp.Title)
}

func TestList_ValidRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	app, _, mockQuerySvc := createTestApp(t, ctrl)

	mockQuerySvc.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(&query.ProviderConfigListResult{
			Items:      []*model.ProviderConfiguration{},
			NextCursor: "",
			HasMore:    false,
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/provider-configurations?limit=10&status=active", nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
