package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"gorm.io/gorm"
)

type SessionRepo interface {
	Create(ctx context.Context, s *model.Session) error
	Delete(ctx context.Context, s *model.Session) error
	Update(ctx context.Context, s *model.Session) error
	Get(ctx context.Context, s *model.Session) (*model.Session, error)
	List(ctx context.Context, projectID uuid.UUID, spaceID *uuid.UUID, notConnected bool) ([]model.Session, error)
	CreateMessageWithAssets(ctx context.Context, msg *model.Message) error
	ListBySessionWithCursor(ctx context.Context, sessionID uuid.UUID, afterCreatedAt time.Time, afterID uuid.UUID, limit int) ([]model.Message, error)
}

type sessionRepo struct{ db *gorm.DB }

func NewSessionRepo(db *gorm.DB) SessionRepo {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, s *model.Session) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *sessionRepo) Delete(ctx context.Context, s *model.Session) error {
	return r.db.WithContext(ctx).Delete(s).Error
}

func (r *sessionRepo) Update(ctx context.Context, s *model.Session) error {
	return r.db.WithContext(ctx).Where(&model.Session{ID: s.ID}).Updates(s).Error
}

func (r *sessionRepo) Get(ctx context.Context, s *model.Session) (*model.Session, error) {
	return s, r.db.WithContext(ctx).Where(&model.Session{ID: s.ID}).First(s).Error
}

func (r *sessionRepo) List(ctx context.Context, projectID uuid.UUID, spaceID *uuid.UUID, notConnected bool) ([]model.Session, error) {
	var sessions []model.Session
	query := r.db.WithContext(ctx).Where(&model.Session{ProjectID: projectID})

	if notConnected {
		query = query.Where("space_id IS NULL")
	} else if spaceID != nil {
		query = query.Where("space_id = ?", spaceID)
	}

	err := query.Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepo) CreateMessageWithAssets(ctx context.Context, msg *model.Message) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First get the message parent id in session
		parent := model.Message{}
		if err := tx.Where(&model.Message{SessionID: msg.SessionID}).Order("created_at desc").Limit(1).Find(&parent).Error; err == nil {
			if parent.ID != uuid.Nil {
				msg.ParentID = &parent.ID
			}
		}

		// Create message
		if err := tx.Create(msg).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *sessionRepo) ListBySessionWithCursor(ctx context.Context, sessionID uuid.UUID, afterCreatedAt time.Time, afterID uuid.UUID, limit int) ([]model.Message, error) {
	q := r.db.WithContext(ctx).Where("session_id = ?", sessionID)

	// Use the (created_at, id) composite cursor; an empty cursor indicates starting from "latest"
	if !afterCreatedAt.IsZero() && afterID != uuid.Nil {
		// Retrieve strictly "older" records (reverse pagination)
		// (created_at, id) < (afterCreatedAt, afterID)
		q = q.Where("(created_at < ?) OR (created_at = ? AND id < ?)", afterCreatedAt, afterCreatedAt, afterID)
	}

	var items []model.Message

	return items, q.Order("created_at DESC, id DESC").Limit(limit).Find(&items).Error
}
