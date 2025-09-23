package service

import (
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/infra/blob"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
)

// MockArtifactRepo is a mock implementation of ArtifactRepo
type MockArtifactRepo struct {
	mock.Mock
}

func (m *MockArtifactRepo) Create(ctx context.Context, a *model.Artifact) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockArtifactRepo) Delete(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) error {
	args := m.Called(ctx, projectID, artifactID)
	return args.Error(0)
}

func (m *MockArtifactRepo) Update(ctx context.Context, a *model.Artifact) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *MockArtifactRepo) GetByID(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) (*model.Artifact, error) {
	args := m.Called(ctx, projectID, artifactID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Artifact), args.Error(1)
}

func (m *MockArtifactRepo) ListByPath(ctx context.Context, projectID uuid.UUID, path string) ([]*model.Artifact, error) {
	args := m.Called(ctx, projectID, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Artifact), args.Error(1)
}

func (m *MockArtifactRepo) ExistsByPathAndFilename(ctx context.Context, projectID uuid.UUID, path string, filename string, excludeID *uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectID, path, filename, excludeID)
	return args.Bool(0), args.Error(1)
}

// S3Interface defines the interface for S3 operations
type S3Interface interface {
	UploadFormFile(ctx context.Context, s3Key string, fileHeader *multipart.FileHeader) (*blob.UploadedMeta, error)
	PresignGet(ctx context.Context, s3Key string, expire time.Duration) (string, error)
}

// MockS3Interface is a mock implementation of S3Interface
type MockS3Interface struct {
	mock.Mock
}

func (m *MockS3Interface) UploadFormFile(ctx context.Context, s3Key string, fileHeader *multipart.FileHeader) (*blob.UploadedMeta, error) {
	args := m.Called(ctx, s3Key, fileHeader)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*blob.UploadedMeta), args.Error(1)
}

func (m *MockS3Interface) PresignGet(ctx context.Context, s3Key string, expire time.Duration) (string, error) {
	args := m.Called(ctx, s3Key, expire)
	return args.String(0), args.Error(1)
}

// testArtifactService is a test version that uses interfaces
type testArtifactService struct {
	r  *MockArtifactRepo
	s3 S3Interface
}

func newTestArtifactService(r *MockArtifactRepo, s3 S3Interface) ArtifactService {
	return &testArtifactService{r: r, s3: s3}
}

func (s *testArtifactService) GetAllPaths(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	return nil, nil
}

func (s *testArtifactService) Create(ctx context.Context, projectID uuid.UUID, path string, fileHeader *multipart.FileHeader, userMeta map[string]interface{}) (*model.Artifact, error) {
	// Check if file with same path and filename already exists in the same project
	exists, err := s.r.ExistsByPathAndFilename(ctx, projectID, path, fileHeader.Filename, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("file already exists")
	}

	// Generate S3 key (simplified like session, path is stored in DB)
	s3Key := "artifacts/" + projectID.String()

	uploadedMeta, err := s.s3.UploadFormFile(ctx, s3Key, fileHeader)
	if err != nil {
		return nil, err
	}

	fileMeta := NewFileMetadataFromUpload(path, fileHeader, uploadedMeta)

	// Create artifact record with separated metadata
	meta := map[string]interface{}{
		"_system": fileMeta.ToSystemMeta(),
	}

	for k, v := range userMeta {
		meta[k] = v
	}

	artifact := &model.Artifact{
		ID:        uuid.New(),
		ProjectID: projectID,
		Meta:      meta,
		AssetMeta: datatypes.NewJSONType(fileMeta.ToAsset()),
	}

	if err := s.r.Create(ctx, artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

func (s *testArtifactService) Delete(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) error {
	if artifactID == (uuid.UUID{}) {
		return errors.New("artifact id is empty")
	}
	return s.r.Delete(ctx, projectID, artifactID)
}

func (s *testArtifactService) GetByID(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) (*model.Artifact, error) {
	if artifactID == (uuid.UUID{}) {
		return nil, errors.New("artifact id is empty")
	}
	return s.r.GetByID(ctx, projectID, artifactID)
}

func (s *testArtifactService) GetPresignedURL(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, expire time.Duration) (string, error) {
	artifact, err := s.GetByID(ctx, projectID, artifactID)
	if err != nil {
		return "", err
	}

	assetData := artifact.AssetMeta.Data()
	if assetData.S3Key == "" {
		return "", errors.New("artifact has no S3 key")
	}

	return s.s3.PresignGet(ctx, assetData.S3Key, expire)
}

func (s *testArtifactService) UpdateFile(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, fileHeader *multipart.FileHeader, newPath *string) (*model.Artifact, error) {
	// Get existing artifact
	artifact, err := s.GetByID(ctx, projectID, artifactID)
	if err != nil {
		return nil, err
	}

	// Determine the target path
	var path string
	if newPath != nil && *newPath != "" {
		path = *newPath
	} else {
		if systemMeta, ok := artifact.Meta["_system"].(map[string]interface{}); ok {
			if pathValue, exists := systemMeta["path"]; exists {
				if pathStr, ok := pathValue.(string); ok {
					path = pathStr
				}
			}
		}
	}

	// Check if file with same path and filename already exists for another artifact in the same project
	exists, err := s.r.ExistsByPathAndFilename(ctx, projectID, path, fileHeader.Filename, &artifactID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("file already exists")
	}

	// Generate new S3 key (simplified like session, path is stored in DB)
	s3Key := "artifacts/" + projectID.String()

	uploadedMeta, err := s.s3.UploadFormFile(ctx, s3Key, fileHeader)
	if err != nil {
		return nil, err
	}

	fileMeta := NewFileMetadataFromUpload(path, fileHeader, uploadedMeta)

	// Update artifact record
	artifact.AssetMeta = datatypes.NewJSONType(fileMeta.ToAsset())

	// Update system meta with new file info
	systemMeta, ok := artifact.Meta["_system"].(map[string]interface{})
	if !ok {
		systemMeta = make(map[string]interface{})
		artifact.Meta["_system"] = systemMeta
	}

	// Update system metadata
	for k, v := range fileMeta.ToSystemMeta() {
		systemMeta[k] = v
	}

	if err := s.r.Update(ctx, artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

func (s *testArtifactService) ListByPath(ctx context.Context, projectID uuid.UUID, path string) ([]*model.Artifact, error) {
	return s.r.ListByPath(ctx, projectID, path)
}

func createTestFileHeader(filename string) *multipart.FileHeader {
	return &multipart.FileHeader{
		Filename: filename,
		Size:     100,
	}
}

func createTestUploadedMeta() *blob.UploadedMeta {
	return &blob.UploadedMeta{
		Bucket: "test-bucket",
		Key:    "test-key",
		ETag:   "test-etag",
		SHA256: "test-sha256",
		MIME:   "text/plain",
		SizeB:  100,
	}
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
				"size":     int64(100),
			},
		},
		AssetMeta: datatypes.NewJSONType(model.Asset{
			Bucket: "test-bucket",
			S3Key:  "test-key",
			ETag:   "test-etag",
			SHA256: "test-sha256",
			MIME:   "text/plain",
			SizeB:  100,
		}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestArtifactService_Create(t *testing.T) {
	projectID := uuid.New()
	fileHeader := createTestFileHeader("test.txt")
	uploadedMeta := createTestUploadedMeta()
	userMeta := map[string]interface{}{
		"description": "test file",
		"category":    "document",
	}

	tests := []struct {
		name        string
		path        string
		userMeta    map[string]interface{}
		setup       func(*MockArtifactRepo, *MockS3Interface)
		expectError bool
		errorMsg    string
	}{
		{
			name:     "successful creation",
			path:     "/",
			userMeta: userMeta,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "test.txt", (*uuid.UUID)(nil)).Return(false, nil)
				s3.On("UploadFormFile", mock.Anything, mock.MatchedBy(func(key string) bool {
					return key == "artifacts/"+projectID.String()
				}), fileHeader).Return(uploadedMeta, nil)
				repo.On("Create", mock.Anything, mock.MatchedBy(func(a *model.Artifact) bool {
					return a.ProjectID == projectID &&
						a.Meta["description"] == "test file" &&
						a.Meta["category"] == "document" &&
						a.Meta["_system"] != nil
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name:     "file already exists",
			path:     "/",
			userMeta: userMeta,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "test.txt", (*uuid.UUID)(nil)).Return(true, nil)
			},
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name:     "check existence error",
			path:     "/",
			userMeta: userMeta,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "test.txt", (*uuid.UUID)(nil)).Return(false, errors.New("db error"))
			},
			expectError: true,
			errorMsg:    "db error",
		},
		{
			name:     "upload error",
			path:     "/",
			userMeta: userMeta,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "test.txt", (*uuid.UUID)(nil)).Return(false, nil)
				s3.On("UploadFormFile", mock.Anything, mock.Anything, fileHeader).Return(nil, errors.New("upload error"))
			},
			expectError: true,
			errorMsg:    "upload error",
		},
		{
			name:     "create record error",
			path:     "/",
			userMeta: userMeta,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "test.txt", (*uuid.UUID)(nil)).Return(false, nil)
				s3.On("UploadFormFile", mock.Anything, mock.Anything, fileHeader).Return(uploadedMeta, nil)
				repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("create error"))
			},
			expectError: true,
			errorMsg:    "create error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockArtifactRepo{}
			mockS3 := &MockS3Interface{}
			tt.setup(mockRepo, mockS3)

			service := newTestArtifactService(mockRepo, mockS3)

			artifact, err := service.Create(context.Background(), projectID, tt.path, fileHeader, tt.userMeta)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, artifact)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, artifact)
				assert.Equal(t, projectID, artifact.ProjectID)
				assert.Equal(t, tt.userMeta["description"], artifact.Meta["description"])
				assert.Equal(t, tt.userMeta["category"], artifact.Meta["category"])
				assert.NotNil(t, artifact.Meta["_system"])
			}

			mockRepo.AssertExpectations(t)
			mockS3.AssertExpectations(t)
		})
	}
}

func TestArtifactService_Delete(t *testing.T) {
	projectID := uuid.New()
	artifactID := uuid.New()

	tests := []struct {
		name        string
		artifactID  uuid.UUID
		setup       func(*MockArtifactRepo)
		expectError bool
		errorMsg    string
	}{
		{
			name:       "successful deletion",
			artifactID: artifactID,
			setup: func(repo *MockArtifactRepo) {
				repo.On("Delete", mock.Anything, projectID, artifactID).Return(nil)
			},
			expectError: false,
		},
		{
			name:        "empty artifact ID",
			artifactID:  uuid.UUID{},
			setup:       func(repo *MockArtifactRepo) {},
			expectError: true,
			errorMsg:    "artifact id is empty",
		},
		{
			name:       "repo error",
			artifactID: artifactID,
			setup: func(repo *MockArtifactRepo) {
				repo.On("Delete", mock.Anything, projectID, artifactID).Return(errors.New("delete error"))
			},
			expectError: true,
			errorMsg:    "delete error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockArtifactRepo{}
			tt.setup(mockRepo)

			service := newTestArtifactService(mockRepo, &MockS3Interface{})

			err := service.Delete(context.Background(), projectID, tt.artifactID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestArtifactService_GetByID(t *testing.T) {
	projectID := uuid.New()
	artifactID := uuid.New()
	artifact := createTestArtifact()
	artifact.ID = artifactID
	artifact.ProjectID = projectID

	tests := []struct {
		name        string
		artifactID  uuid.UUID
		setup       func(*MockArtifactRepo)
		expectError bool
		errorMsg    string
	}{
		{
			name:       "successful get",
			artifactID: artifactID,
			setup: func(repo *MockArtifactRepo) {
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(artifact, nil)
			},
			expectError: false,
		},
		{
			name:        "empty artifact ID",
			artifactID:  uuid.UUID{},
			setup:       func(repo *MockArtifactRepo) {},
			expectError: true,
			errorMsg:    "artifact id is empty",
		},
		{
			name:       "repo error",
			artifactID: artifactID,
			setup: func(repo *MockArtifactRepo) {
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(nil, errors.New("get error"))
			},
			expectError: true,
			errorMsg:    "get error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockArtifactRepo{}
			tt.setup(mockRepo)

			service := newTestArtifactService(mockRepo, &MockS3Interface{})

			result, err := service.GetByID(context.Background(), projectID, tt.artifactID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, artifactID, result.ID)
				assert.Equal(t, projectID, result.ProjectID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestArtifactService_GetPresignedURL(t *testing.T) {
	projectID := uuid.New()
	artifactID := uuid.New()
	artifact := createTestArtifact()
	artifact.ID = artifactID
	artifact.ProjectID = projectID

	tests := []struct {
		name        string
		artifactID  uuid.UUID
		expire      time.Duration
		setup       func(*MockArtifactRepo, *MockS3Interface)
		expectError bool
		errorMsg    string
		expectedURL string
	}{
		{
			name:       "successful presigned URL",
			artifactID: artifactID,
			expire:     time.Hour,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(artifact, nil)
				s3.On("PresignGet", mock.Anything, "test-key", time.Hour).Return("https://example.com/presigned-url", nil)
			},
			expectError: false,
			expectedURL: "https://example.com/presigned-url",
		},
		{
			name:       "get artifact error",
			artifactID: artifactID,
			expire:     time.Hour,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(nil, errors.New("get error"))
			},
			expectError: true,
			errorMsg:    "get error",
		},
		{
			name:       "no S3 key",
			artifactID: artifactID,
			expire:     time.Hour,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				artifactNoS3Key := *artifact
				artifactNoS3Key.AssetMeta = datatypes.NewJSONType(model.Asset{})
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(&artifactNoS3Key, nil)
			},
			expectError: true,
			errorMsg:    "artifact has no S3 key",
		},
		{
			name:       "presign error",
			artifactID: artifactID,
			expire:     time.Hour,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(artifact, nil)
				s3.On("PresignGet", mock.Anything, "test-key", time.Hour).Return("", errors.New("presign error"))
			},
			expectError: true,
			errorMsg:    "presign error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockArtifactRepo{}
			mockS3 := &MockS3Interface{}
			tt.setup(mockRepo, mockS3)

			service := newTestArtifactService(mockRepo, mockS3)

			url, err := service.GetPresignedURL(context.Background(), projectID, tt.artifactID, tt.expire)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}

			mockRepo.AssertExpectations(t)
			mockS3.AssertExpectations(t)
		})
	}
}

func TestArtifactService_UpdateFile(t *testing.T) {
	projectID := uuid.New()
	artifactID := uuid.New()
	fileHeader := createTestFileHeader("updated.txt")
	uploadedMeta := createTestUploadedMeta()

	tests := []struct {
		name        string
		artifactID  uuid.UUID
		newPath     *string
		setup       func(*MockArtifactRepo, *MockS3Interface)
		expectError bool
		errorMsg    string
	}{
		{
			name:       "successful update with new path",
			artifactID: artifactID,
			newPath:    stringPtr("/updated"),
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				existingArtifact := createTestArtifact()
				existingArtifact.ID = artifactID
				existingArtifact.ProjectID = projectID
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(existingArtifact, nil)
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/updated", "updated.txt", &artifactID).Return(false, nil)
				s3.On("UploadFormFile", mock.Anything, mock.Anything, fileHeader).Return(uploadedMeta, nil)
				repo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Artifact) bool {
					return a.ID == artifactID && a.ProjectID == projectID
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name:       "successful update without new path",
			artifactID: artifactID,
			newPath:    nil,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				existingArtifact := createTestArtifact()
				existingArtifact.ID = artifactID
				existingArtifact.ProjectID = projectID
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(existingArtifact, nil)
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "updated.txt", &artifactID).Return(false, nil)
				s3.On("UploadFormFile", mock.Anything, mock.Anything, fileHeader).Return(uploadedMeta, nil)
				repo.On("Update", mock.Anything, mock.MatchedBy(func(a *model.Artifact) bool {
					return a.ID == artifactID && a.ProjectID == projectID
				})).Return(nil)
			},
			expectError: false,
		},
		{
			name:       "get artifact error",
			artifactID: artifactID,
			newPath:    nil,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(nil, errors.New("get error"))
			},
			expectError: true,
			errorMsg:    "get error",
		},
		{
			name:       "file already exists",
			artifactID: artifactID,
			newPath:    stringPtr("/updated"),
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				existingArtifact := createTestArtifact()
				existingArtifact.ID = artifactID
				existingArtifact.ProjectID = projectID
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(existingArtifact, nil)
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/updated", "updated.txt", &artifactID).Return(true, nil)
			},
			expectError: true,
			errorMsg:    "already exists",
		},
		{
			name:       "upload error",
			artifactID: artifactID,
			newPath:    nil,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				existingArtifact := createTestArtifact()
				existingArtifact.ID = artifactID
				existingArtifact.ProjectID = projectID
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(existingArtifact, nil)
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "updated.txt", &artifactID).Return(false, nil)
				s3.On("UploadFormFile", mock.Anything, mock.Anything, fileHeader).Return(nil, errors.New("upload error"))
			},
			expectError: true,
			errorMsg:    "upload error",
		},
		{
			name:       "update error",
			artifactID: artifactID,
			newPath:    nil,
			setup: func(repo *MockArtifactRepo, s3 *MockS3Interface) {
				existingArtifact := createTestArtifact()
				existingArtifact.ID = artifactID
				existingArtifact.ProjectID = projectID
				repo.On("GetByID", mock.Anything, projectID, artifactID).Return(existingArtifact, nil)
				repo.On("ExistsByPathAndFilename", mock.Anything, projectID, "/", "updated.txt", &artifactID).Return(false, nil)
				s3.On("UploadFormFile", mock.Anything, mock.Anything, fileHeader).Return(uploadedMeta, nil)
				repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update error"))
			},
			expectError: true,
			errorMsg:    "update error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockArtifactRepo{}
			mockS3 := &MockS3Interface{}
			tt.setup(mockRepo, mockS3)

			service := newTestArtifactService(mockRepo, mockS3)

			artifact, err := service.UpdateFile(context.Background(), projectID, tt.artifactID, fileHeader, tt.newPath)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, artifact)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, artifact)
				assert.Equal(t, artifactID, artifact.ID)
				assert.Equal(t, projectID, artifact.ProjectID)
			}

			mockRepo.AssertExpectations(t)
			mockS3.AssertExpectations(t)
		})
	}
}

func TestArtifactService_ListByPath(t *testing.T) {
	projectID := uuid.New()
	artifacts := []*model.Artifact{
		createTestArtifact(),
		createTestArtifact(),
	}

	tests := []struct {
		name        string
		path        string
		setup       func(*MockArtifactRepo)
		expectError bool
		errorMsg    string
		expectedLen int
	}{
		{
			name: "successful list",
			path: "/test",
			setup: func(repo *MockArtifactRepo) {
				repo.On("ListByPath", mock.Anything, projectID, "/test").Return(artifacts, nil)
			},
			expectError: false,
			expectedLen: 2,
		},
		{
			name: "repo error",
			path: "/test",
			setup: func(repo *MockArtifactRepo) {
				repo.On("ListByPath", mock.Anything, projectID, "/test").Return(nil, errors.New("list error"))
			},
			expectError: true,
			errorMsg:    "list error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockArtifactRepo{}
			tt.setup(mockRepo)

			service := newTestArtifactService(mockRepo, &MockS3Interface{})

			result, err := service.ListByPath(context.Background(), projectID, tt.path)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedLen)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestFileMetadata_ToAsset(t *testing.T) {
	fm := &FileMetadata{
		Bucket: "test-bucket",
		S3Key:  "test-key",
		ETag:   "test-etag",
		SHA256: "test-sha256",
		MIME:   "text/plain",
		SizeB:  100,
	}

	asset := fm.ToAsset()

	assert.Equal(t, "test-bucket", asset.Bucket)
	assert.Equal(t, "test-key", asset.S3Key)
	assert.Equal(t, "test-etag", asset.ETag)
	assert.Equal(t, "test-sha256", asset.SHA256)
	assert.Equal(t, "text/plain", asset.MIME)
	assert.Equal(t, int64(100), asset.SizeB)
}

func TestFileMetadata_ToSystemMeta(t *testing.T) {
	fm := &FileMetadata{
		Path:     "/test",
		Filename: "test.txt",
		MIME:     "text/plain",
		SizeB:    100,
	}

	meta := fm.ToSystemMeta()

	assert.Equal(t, "/test", meta["path"])
	assert.Equal(t, "test.txt", meta["filename"])
	assert.Equal(t, "text/plain", meta["mime"])
	assert.Equal(t, int64(100), meta["size"])
}

func TestNewFileMetadataFromUpload(t *testing.T) {
	path := "/test"
	fileHeader := createTestFileHeader("test.txt")
	uploadedMeta := createTestUploadedMeta()

	fm := NewFileMetadataFromUpload(path, fileHeader, uploadedMeta)

	assert.Equal(t, "/test", fm.Path)
	assert.Equal(t, "test.txt", fm.Filename)
	assert.Equal(t, "text/plain", fm.MIME)
	assert.Equal(t, int64(100), fm.SizeB)
	assert.Equal(t, "test-bucket", fm.Bucket)
	assert.Equal(t, "test-key", fm.S3Key)
	assert.Equal(t, "test-etag", fm.ETag)
	assert.Equal(t, "test-sha256", fm.SHA256)
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
