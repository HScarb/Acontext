package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/infra/httpclient"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"github.com/stretchr/testify/assert"
)

func setupToolRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// getMockToolCoreClient returns a mock CoreClient for testing
func getMockToolCoreClient() *httpclient.CoreClient {
	// Create a minimal CoreClient with invalid URL
	// This will cause network errors when called, which is expected in tests
	return &httpclient.CoreClient{
		BaseURL:    "http://invalid-test-url:99999",
		HTTPClient: &http.Client{},
	}
}

func TestToolHandler_RenameToolName(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{} // Changed to interface{} to allow invalid JSON
		expectedStatus int
		skip           bool // Skip tests that require Core service
	}{
		{
			name: "successful rename",
			requestBody: RenameToolNameReq{
				Rename: []ToolRenameItem{
					{OldName: "old_tool", NewName: "new_tool"},
				},
			},
			expectedStatus: http.StatusInternalServerError, // Will fail without real core service
			skip:           true,                           // Requires Core service integration
		},
		{
			name:           "empty rename list",
			requestBody:    RenameToolNameReq{Rename: []ToolRenameItem{}},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing old_name",
			requestBody: map[string]interface{}{
				"rename": []map[string]string{
					{"new_name": "new_tool"},
				},
			},
			expectedStatus: http.StatusInternalServerError, // Binding passes but core client fails
			skip:           true,                           // This requires core service to test properly
		},
		{
			name: "missing new_name",
			requestBody: map[string]interface{}{
				"rename": []map[string]string{
					{"old_name": "old_tool"},
				},
			},
			expectedStatus: http.StatusInternalServerError, // Binding passes but core client fails
			skip:           true,                           // This requires core service to test properly
		},
		{
			name:           "invalid JSON",
			requestBody:    "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Skipping test that requires Core service integration")
			}

			handler := NewToolHandler(getMockToolCoreClient())
			router := setupToolRouter()
			// Add middleware to set project in context
			router.Use(func(c *gin.Context) {
				c.Set("project", &model.Project{ID: uuid.New()})
				c.Next()
			})
			router.PUT("/tool/name", handler.RenameToolName)

			var body []byte
			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = sonic.Marshal(tt.requestBody)
			}
			req := httptest.NewRequest("PUT", "/tool/name", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestToolHandler_GetToolName(t *testing.T) {
	t.Run("successful get tool names", func(t *testing.T) {
		t.Skip("Skipping test that requires Core service integration")

		handler := NewToolHandler(getMockToolCoreClient())
		router := setupToolRouter()

		router.Use(func(c *gin.Context) {
			c.Set("project", &model.Project{ID: uuid.New()})
			c.Next()
		})

		router.GET("/tool/name", handler.GetToolName)

		req := httptest.NewRequest("GET", "/tool/name", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Will fail without real core service
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
