package repo

import (
	"context"

	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"gorm.io/gorm"
)

type ArtifactRepo interface {
	Create(ctx context.Context, a *model.Artifact) error
	Delete(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) error
	Update(ctx context.Context, a *model.Artifact) error
	GetByID(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) (*model.Artifact, error)
	ListByPath(ctx context.Context, projectID uuid.UUID, path string) ([]*model.Artifact, error)
	GetAllPaths(ctx context.Context, projectID uuid.UUID) ([]string, error)
	ExistsByPathAndFilename(ctx context.Context, projectID uuid.UUID, path string, filename string, excludeID *uuid.UUID) (bool, error)
}

type artifactRepo struct{ db *gorm.DB }

func NewArtifactRepo(db *gorm.DB) ArtifactRepo {
	return &artifactRepo{db: db}
}

func (r *artifactRepo) Create(ctx context.Context, a *model.Artifact) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *artifactRepo) Delete(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ? AND project_id = ?", artifactID, projectID).Delete(&model.Artifact{}).Error
}

func (r *artifactRepo) Update(ctx context.Context, a *model.Artifact) error {
	return r.db.WithContext(ctx).Where("id = ? AND project_id = ?", a.ID, a.ProjectID).Updates(a).Error
}

func (r *artifactRepo) GetByID(ctx context.Context, projectID uuid.UUID, artifactID uuid.UUID) (*model.Artifact, error) {
	var artifact model.Artifact
	err := r.db.WithContext(ctx).Where("id = ? AND project_id = ?", artifactID, projectID).First(&artifact).Error
	if err != nil {
		return nil, err
	}
	return &artifact, nil
}

func (r *artifactRepo) ListByPath(ctx context.Context, projectID uuid.UUID, path string) ([]*model.Artifact, error) {
	var artifacts []*model.Artifact
	query := r.db.WithContext(ctx).Where("project_id = ?", projectID)

	// If path is specified, filter by path in _system meta
	if path != "" {
		query = query.Where("meta->'_system'->>'path' = ?", path)
	}

	err := query.Find(&artifacts).Error
	if err != nil {
		return nil, err
	}
	return artifacts, nil
}

func (r *artifactRepo) GetAllPaths(ctx context.Context, projectID uuid.UUID) ([]string, error) {
	var paths []string
	err := r.db.WithContext(ctx).
		Model(&model.Artifact{}).
		Where("project_id = ?", projectID).
		Distinct("meta->'_system'->>'path'").
		Pluck("meta->'_system'->>'path'", &paths).Error
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func (r *artifactRepo) ExistsByPathAndFilename(ctx context.Context, projectID uuid.UUID, path string, filename string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&model.Artifact{}).
		Where("project_id = ? AND meta->'_system'->>'path' = ? AND meta->'_system'->>'filename' = ?",
			projectID, path, filename)

	// Exclude specific artifact ID (useful for update operations)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
