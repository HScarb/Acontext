package handler

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
)

// MockArtifactService is a mock implementation of ArtifactService
type MockArtifactService struct {
	mock.Mock
}

func (m *MockArtifactService) Create(ctx context.Context, projectID uuid.UUID, path string, fileHeader *multipart.FileHeader, userMeta map[string]interface{}) (*model.Artifact, error) {
	args := m.Called(ctx, projectID, path, fileHeader, userMeta)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Artifact), args.Error(1)
}

func (m *MockArtifactService) Delete(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) error {
	args := m.Called(ctx, projectID, artifactID)
	return args.Error(0)
}

func (m *MockArtifactService) GetByID(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) (*model.Artifact, error) {
	args := m.Called(ctx, projectID, artifactID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Artifact), args.Error(1)
}

func (m *MockArtifactService) GetPresignedURL(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, expire time.Duration) (string, error) {
	args := m.Called(ctx, projectID, artifactID, expire)
	return args.String(0), args.Error(1)
}

func (m *MockArtifactService) UpdateFile(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, fileHeader *multipart.FileHeader, newPath *string) (*model.Artifact, error) {
	args := m.Called(ctx, projectID, artifactID, fileHeader, newPath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Artifact), args.Error(1)
}

func (m *MockArtifactService) ListByPath(ctx context.Context, projectID uuid.UUID, path string) ([]*model.Artifact, error) {
	args := m.Called(ctx, projectID, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Artifact), args.Error(1)
}

func (m *MockArtifactService) GetAllPaths(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func setupArtifactRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestArtifact() *model.Artifact {
	projectID := uuid.New()
	artifactID := uuid.New()

	return &model.Artifact{
		ID:        artifactID,
		ProjectID: projectID,
		Meta: datatypes.JSONMap{
			"_system": map[string]interface{}{
				"path":     "/",
				"filename": "test.txt",
				"mime":     "text/plain",
				"size":     int64(10),
			},
		},
		AssetMeta: datatypes.NewJSONType(model.Asset{
			Bucket: "test-bucket",
			S3Key:  "artifacts/test-key",
			ETag:   "test-etag",
			SHA256: "test-sha256",
			MIME:   "text/plain",
			SizeB:  10,
		}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestArtifactHandler_CreateArtifact(t *testing.T) {
	projectID := uuid.New()
	artifact := createTestArtifact()
	artifact.ProjectID = projectID

	tests := []struct {
		name           string
		requestBody    map[string]string
		fileContent    string
		fileName       string
		setup          func(*MockArtifactService)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful artifact creation",
			requestBody: map[string]string{
				"path": "/",
				"meta": `{"description": "test file"}`,
			},
			fileContent: "test content",
			fileName:    "test.txt",
			setup: func(svc *MockArtifactService) {
				svc.On("Create", mock.Anything, projectID, "/", mock.Anything, mock.MatchedBy(func(meta map[string]interface{}) bool {
					return meta["description"] == "test file"
				})).Return(artifact, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "successful artifact creation with default path",
			requestBody: map[string]string{
				"meta": `{"description": "test file"}`,
			},
			fileContent: "test content",
			fileName:    "test.txt",
			setup: func(svc *MockArtifactService) {
				svc.On("Create", mock.Anything, projectID, "/", mock.Anything, mock.MatchedBy(func(meta map[string]interface{}) bool {
					return meta["description"] == "test file"
				})).Return(artifact, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid path",
			requestBody: map[string]string{
				"path": "../invalid",
			},
			fileContent:    "test content",
			fileName:       "test.txt",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid path",
		},
		{
			name: "missing file",
			requestBody: map[string]string{
				"path": "/",
			},
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "file is required",
		},
		{
			name: "invalid meta JSON",
			requestBody: map[string]string{
				"path": "/",
				"meta": "invalid json",
			},
			fileContent:    "test content",
			fileName:       "test.txt",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid meta JSON format",
		},
		{
			name: "reserved key in meta",
			requestBody: map[string]string{
				"path": "/",
				"meta": `{"_system": "reserved"}`,
			},
			fileContent:    "test content",
			fileName:       "test.txt",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "reserved key",
		},
		{
			name: "service error",
			requestBody: map[string]string{
				"path": "/",
			},
			fileContent: "test content",
			fileName:    "test.txt",
			setup: func(svc *MockArtifactService) {
				svc.On("Create", mock.Anything, projectID, "/", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockArtifactService{}
			tt.setup(mockService)
			handler := NewArtifactHandler(mockService)

			router := setupArtifactRouter()
			router.POST("/artifact", func(c *gin.Context) {
				c.Set("project", &model.Project{ID: projectID})
				handler.CreateArtifact(c)
			})

			// Create multipart form data
			formBody := &bytes.Buffer{}
			writer := multipart.NewWriter(formBody)

			// Add file
			if tt.fileName != "" {
				part, _ := writer.CreateFormFile("file", tt.fileName)
				part.Write([]byte(tt.fileContent))
			}

			// Add form fields
			for key, value := range tt.requestBody {
				writer.WriteField(key, value)
			}
			writer.Close()

			req := httptest.NewRequest("POST", "/artifact", formBody)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := sonic.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				if response["message"] != nil {
					assert.Contains(t, response["message"], tt.expectedError)
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestArtifactHandler_DeleteArtifact(t *testing.T) {
	projectID := uuid.New()
	artifactID := uuid.New()

	tests := []struct {
		name           string
		artifactID     string
		setup          func(*MockArtifactService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "successful deletion",
			artifactID: artifactID.String(),
			setup: func(svc *MockArtifactService) {
				svc.On("Delete", mock.Anything, projectID, artifactID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid artifact ID",
			artifactID:     "invalid-uuid",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid UUID",
		},
		{
			name:       "service error",
			artifactID: artifactID.String(),
			setup: func(svc *MockArtifactService) {
				svc.On("Delete", mock.Anything, projectID, artifactID).Return(errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockArtifactService{}
			tt.setup(mockService)
			handler := NewArtifactHandler(mockService)

			router := setupArtifactRouter()
			router.DELETE("/artifact/:artifact_id", func(c *gin.Context) {
				c.Set("project", &model.Project{ID: projectID})
				handler.DeleteArtifact(c)
			})

			req := httptest.NewRequest("DELETE", "/artifact/"+tt.artifactID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := sonic.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				if response["message"] != nil {
					assert.Contains(t, response["message"], tt.expectedError)
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestArtifactHandler_GetArtifact(t *testing.T) {
	projectID := uuid.New()
	artifactID := uuid.New()
	artifact := createTestArtifact()
	artifact.ID = artifactID
	artifact.ProjectID = projectID

	tests := []struct {
		name           string
		artifactID     string
		queryParams    string
		setup          func(*MockArtifactService)
		expectedStatus int
		expectedError  string
		expectURL      bool
	}{
		{
			name:        "successful get without URL",
			artifactID:  artifactID.String(),
			queryParams: "",
			setup: func(svc *MockArtifactService) {
				svc.On("GetByID", mock.Anything, projectID, artifactID).Return(artifact, nil)
			},
			expectedStatus: http.StatusOK,
			expectURL:      false,
		},
		{
			name:        "successful get with URL",
			artifactID:  artifactID.String(),
			queryParams: "?with_public_url=true&expire=7200",
			setup: func(svc *MockArtifactService) {
				svc.On("GetByID", mock.Anything, projectID, artifactID).Return(artifact, nil)
				svc.On("GetPresignedURL", mock.Anything, projectID, artifactID, 2*time.Hour).Return("https://example.com/presigned-url", nil)
			},
			expectedStatus: http.StatusOK,
			expectURL:      true,
		},
		{
			name:           "invalid artifact ID",
			artifactID:     "invalid-uuid",
			queryParams:    "",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid UUID",
		},
		{
			name:        "service error",
			artifactID:  artifactID.String(),
			queryParams: "",
			setup: func(svc *MockArtifactService) {
				svc.On("GetByID", mock.Anything, projectID, artifactID).Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:        "presigned URL error",
			artifactID:  artifactID.String(),
			queryParams: "?with_public_url=true",
			setup: func(svc *MockArtifactService) {
				svc.On("GetByID", mock.Anything, projectID, artifactID).Return(artifact, nil)
				svc.On("GetPresignedURL", mock.Anything, projectID, artifactID, 1*time.Hour).Return("", errors.New("URL error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockArtifactService{}
			tt.setup(mockService)
			handler := NewArtifactHandler(mockService)

			router := setupArtifactRouter()
			router.GET("/artifact/:artifact_id", func(c *gin.Context) {
				c.Set("project", &model.Project{ID: projectID})
				handler.GetArtifact(c)
			})

			req := httptest.NewRequest("GET", "/artifact/"+tt.artifactID+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := sonic.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				if response["message"] != nil {
					assert.Contains(t, response["message"], tt.expectedError)
				}
			} else if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := sonic.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				if tt.expectURL {
					data := response["data"].(map[string]interface{})
					assert.NotNil(t, data["public_url"])
				} else {
					data := response["data"].(map[string]interface{})
					assert.Nil(t, data["public_url"])
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestArtifactHandler_UpdateArtifact(t *testing.T) {
	projectID := uuid.New()
	artifactID := uuid.New()
	artifact := createTestArtifact()
	artifact.ID = artifactID
	artifact.ProjectID = projectID

	tests := []struct {
		name           string
		artifactID     string
		requestBody    map[string]string
		fileContent    string
		fileName       string
		setup          func(*MockArtifactService)
		expectedStatus int
		expectedError  string
	}{
		{
			name:       "successful update",
			artifactID: artifactID.String(),
			requestBody: map[string]string{
				"path": "updated",
			},
			fileContent: "updated content",
			fileName:    "updated.txt",
			setup: func(svc *MockArtifactService) {
				svc.On("UpdateFile", mock.MatchedBy(func(ctx context.Context) bool { return true }), projectID, artifactID, mock.MatchedBy(func(file *multipart.FileHeader) bool { return true }), mock.MatchedBy(func(path *string) bool {
					return path != nil && *path == "updated"
				})).Return(artifact, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "successful update without path",
			artifactID:  artifactID.String(),
			requestBody: map[string]string{},
			fileContent: "updated content",
			fileName:    "updated.txt",
			setup: func(svc *MockArtifactService) {
				svc.On("UpdateFile", mock.MatchedBy(func(ctx context.Context) bool { return true }), projectID, artifactID, mock.MatchedBy(func(file *multipart.FileHeader) bool { return true }), mock.Anything).Return(artifact, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "invalid path",
			artifactID: artifactID.String(),
			requestBody: map[string]string{
				"path": "../invalid",
			},
			fileContent:    "updated content",
			fileName:       "updated.txt",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid path",
		},
		{
			name:           "invalid artifact ID",
			artifactID:     "invalid-uuid",
			requestBody:    map[string]string{},
			fileContent:    "updated content",
			fileName:       "updated.txt",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid UUID",
		},
		{
			name:           "missing file",
			artifactID:     artifactID.String(),
			requestBody:    map[string]string{},
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "file is required",
		},
		{
			name:        "service error",
			artifactID:  artifactID.String(),
			requestBody: map[string]string{},
			fileContent: "updated content",
			fileName:    "updated.txt",
			setup: func(svc *MockArtifactService) {
				svc.On("UpdateFile", mock.Anything, projectID, artifactID, mock.Anything, mock.Anything).Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockArtifactService{}
			tt.setup(mockService)
			handler := NewArtifactHandler(mockService)

			router := setupArtifactRouter()
			router.PUT("/artifact/:artifact_id", func(c *gin.Context) {
				c.Set("project", &model.Project{ID: projectID})
				handler.UpdateArtifact(c)
			})

			// Create multipart form data
			formBody := &bytes.Buffer{}
			writer := multipart.NewWriter(formBody)

			// Add file
			if tt.fileName != "" {
				part, _ := writer.CreateFormFile("file", tt.fileName)
				part.Write([]byte(tt.fileContent))
			}

			// Add form fields
			for key, value := range tt.requestBody {
				writer.WriteField(key, value)
			}
			writer.Close()

			req := httptest.NewRequest("PUT", "/artifact/"+tt.artifactID, formBody)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := sonic.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				if response["message"] != nil {
					assert.Contains(t, response["message"], tt.expectedError)
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestArtifactHandler_ListArtifacts(t *testing.T) {
	projectID := uuid.New()
	artifacts := []*model.Artifact{
		createTestArtifact(),
		createTestArtifact(),
	}

	tests := []struct {
		name           string
		queryParams    string
		setup          func(*MockArtifactService)
		expectedStatus int
		expectedError  string
		expectedCount  int
	}{
		{
			name:        "successful list with path",
			queryParams: "?path=/test",
			setup: func(svc *MockArtifactService) {
				svc.On("ListByPath", mock.Anything, projectID, "/test").Return(artifacts, nil)
				svc.On("GetAllPaths", mock.Anything, projectID).Return([]string{"/test/subdir1/file1.txt", "/test/subdir2/file2.txt"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:        "successful list with default path",
			queryParams: "",
			setup: func(svc *MockArtifactService) {
				svc.On("ListByPath", mock.Anything, projectID, "/").Return(artifacts, nil)
				svc.On("GetAllPaths", mock.Anything, projectID).Return([]string{"/documents/file1.txt", "/images/photo.jpg", "/webp/image1.webp"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "invalid path",
			queryParams:    "?path=../invalid",
			setup:          func(svc *MockArtifactService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid path",
		},
		{
			name:        "service error",
			queryParams: "",
			setup: func(svc *MockArtifactService) {
				svc.On("ListByPath", mock.Anything, projectID, "/").Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockArtifactService{}
			tt.setup(mockService)
			handler := NewArtifactHandler(mockService)

			router := setupArtifactRouter()
			router.GET("/artifact", func(c *gin.Context) {
				c.Set("project", &model.Project{ID: projectID})
				handler.ListArtifacts(c)
			})

			req := httptest.NewRequest("GET", "/artifact"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := sonic.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				if response["message"] != nil {
					assert.Contains(t, response["message"], tt.expectedError)
				}
			} else if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := sonic.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				if response["data"] != nil {
					data := response["data"].(map[string]interface{})
					// Check that artifacts array has expected count
					if artifacts, ok := data["artifacts"].([]interface{}); ok {
						assert.Equal(t, tt.expectedCount, len(artifacts))
					}
					// Check that directories array exists
					assert.Contains(t, data, "directories")
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}
