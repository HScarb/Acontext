package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/infra/blob"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"github.com/memodb-io/Acontext/internal/modules/repo"
	"gorm.io/datatypes"
)

// FileMetadata centrally manages file-related metadata
type FileMetadata struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
	MIME     string `json:"mime"`
	SizeB    int64  `json:"size_b"`
	Bucket   string `json:"bucket"`
	S3Key    string `json:"s3_key"`
	ETag     string `json:"etag"`
	SHA256   string `json:"sha256"`
}

// ToAsset converts to Asset model
func (fm *FileMetadata) ToAsset() model.Asset {
	return model.Asset{
		Bucket: fm.Bucket,
		S3Key:  fm.S3Key,
		ETag:   fm.ETag,
		SHA256: fm.SHA256,
		MIME:   fm.MIME,
		SizeB:  fm.SizeB,
	}
}

// ToSystemMeta converts to system metadata
func (fm *FileMetadata) ToSystemMeta() map[string]interface{} {
	return map[string]interface{}{
		"path":     fm.Path,
		"filename": fm.Filename,
		"mime":     fm.MIME,
		"size":     fm.SizeB,
	}
}

// NewFileMetadataFromUpload creates FileMetadata from the uploaded file
func NewFileMetadataFromUpload(path string, fileHeader *multipart.FileHeader, uploadedMeta *blob.UploadedMeta) *FileMetadata {
	return &FileMetadata{
		Path:     path,
		Filename: fileHeader.Filename,
		MIME:     uploadedMeta.MIME,
		SizeB:    uploadedMeta.SizeB,
		Bucket:   uploadedMeta.Bucket,
		S3Key:    uploadedMeta.Key,
		ETag:     uploadedMeta.ETag,
		SHA256:   uploadedMeta.SHA256,
	}
}

type ArtifactService interface {
	Create(ctx context.Context, projectID uuid.UUID, path string, fileHeader *multipart.FileHeader, userMeta map[string]interface{}) (*model.Artifact, error)
	Delete(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) error
	GetByID(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) (*model.Artifact, error)
	GetPresignedURL(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, expire time.Duration) (string, error)
	UpdateFile(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, fileHeader *multipart.FileHeader, newPath *string) (*model.Artifact, error)
	ListByPath(ctx context.Context, projectID uuid.UUID, path string) ([]*model.Artifact, error)
	GetAllPaths(ctx context.Context, projectID uuid.UUID) ([]string, error)
}

type artifactService struct {
	r  repo.ArtifactRepo
	s3 *blob.S3Deps
}

func NewArtifactService(r repo.ArtifactRepo, s3 *blob.S3Deps) ArtifactService {
	return &artifactService{r: r, s3: s3}
}

func (s *artifactService) Create(ctx context.Context, projectID uuid.UUID, path string, fileHeader *multipart.FileHeader, userMeta map[string]interface{}) (*model.Artifact, error) {
	// Check if file with same path and filename already exists in the same project
	exists, err := s.r.ExistsByPathAndFilename(ctx, projectID, path, fileHeader.Filename, nil)
	if err != nil {
		return nil, fmt.Errorf("check file existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("file '%s' already exists in path '%s'", fileHeader.Filename, path)
	}

	// Generate S3 key (simplified like session, path is stored in DB)
	s3Key := fmt.Sprintf("artifacts/%s", projectID.String())

	uploadedMeta, err := s.s3.UploadFormFile(ctx, s3Key, fileHeader)
	if err != nil {
		return nil, fmt.Errorf("upload file to S3: %w", err)
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
		return nil, fmt.Errorf("create artifact record: %w", err)
	}

	return artifact, nil
}

func (s *artifactService) Delete(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) error {
	if len(artifactID) == 0 {
		return errors.New("artifact id is empty")
	}
	return s.r.Delete(ctx, projectID, artifactID)
}

func (s *artifactService) GetByID(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) (*model.Artifact, error) {
	if len(artifactID) == 0 {
		return nil, errors.New("artifact id is empty")
	}
	return s.r.GetByID(ctx, projectID, artifactID)
}

func (s *artifactService) GetPresignedURL(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, expire time.Duration) (string, error) {
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

func (s *artifactService) UpdateFile(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID, fileHeader *multipart.FileHeader, newPath *string) (*model.Artifact, error) {
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
		return nil, fmt.Errorf("check file existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("file '%s' already exists in path '%s'", fileHeader.Filename, path)
	}

	// Generate new S3 key (simplified like session, path is stored in DB)
	s3Key := fmt.Sprintf("artifacts/%s", projectID.String())

	uploadedMeta, err := s.s3.UploadFormFile(ctx, s3Key, fileHeader)
	if err != nil {
		return nil, fmt.Errorf("upload file to S3: %w", err)
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

	// 使用统一的方法更新系统元数据
	for k, v := range fileMeta.ToSystemMeta() {
		systemMeta[k] = v
	}

	if err := s.r.Update(ctx, artifact); err != nil {
		return nil, fmt.Errorf("update artifact record: %w", err)
	}

	return artifact, nil
}

func (s *artifactService) ListByPath(ctx context.Context, projectID uuid.UUID, path string) ([]*model.Artifact, error) {
	return s.r.ListByPath(ctx, projectID, path)
}

func (s *artifactService) GetAllPaths(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	return s.r.GetAllPaths(ctx, projectID)
}
