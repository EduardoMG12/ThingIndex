package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"thing-index/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestMetaHandlerGetRuntimeProfile(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := NewMetaHandler()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/meta/runtime-profile", nil)

	handler.GetRuntimeProfile(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	data, ok := payload.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected response data map, got %#v", payload.Data)
	}

	if data["product_name"] != "ThingIndex" {
		t.Fatalf("expected product_name ThingIndex, got %#v", data["product_name"])
	}

	targetUsers, ok := data["target_users"].([]any)
	if !ok || len(targetUsers) < 2 {
		t.Fatalf("expected target_users payload, got %#v", data["target_users"])
	}

	artifacts, ok := data["artifact_refs"].([]any)
	if !ok || len(artifacts) < 2 {
		t.Fatalf("expected artifact refs payload, got %#v", data["artifact_refs"])
	}
}
