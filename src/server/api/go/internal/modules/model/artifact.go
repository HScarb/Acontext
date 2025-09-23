package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Artifact struct {
	ID        uuid.UUID                 `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProjectID uuid.UUID                 `gorm:"type:uuid;not null;index" json:"project_id"`
	Meta      datatypes.JSONMap         `gorm:"type:jsonb" swaggertype:"object" json:"meta"`
	AssetMeta datatypes.JSONType[Asset] `gorm:"type:jsonb;not null" swaggertype:"-" json:"-"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Artifact <-> Project
	Project *Project `gorm:"foreignKey:ProjectID;references:ID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;" json:"project"`
}

func (Artifact) TableName() string { return "artifacts" }
