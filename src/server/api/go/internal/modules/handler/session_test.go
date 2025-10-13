package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"github.com/memodb-io/Acontext/internal/modules/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
)

// MockSessionService is a mock implementation of SessionService
type MockSessionService struct {
	mock.Mock
}

func (m *MockSessionService) Create(ctx context.Context, s *model.Session) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSessionService) Delete(ctx context.Context, projectID uuid.UUID, sessionID uuid.UUID) error {
	args := m.Called(ctx, projectID, sessionID)
	return args.Error(0)
}

func (m *MockSessionService) UpdateByID(ctx context.Context, s *model.Session) error {
	args := m.Called(ctx, s)
	return args.Error(0)
}

func (m *MockSessionService) GetByID(ctx context.Context, s *model.Session) (*model.Session, error) {
	args := m.Called(ctx, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Session), args.Error(1)
}

func (m *MockSessionService) SendMessage(ctx context.Context, in service.SendMessageInput) (*model.Message, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Message), args.Error(1)
}

func (m *MockSessionService) GetMessages(ctx context.Context, in service.GetMessagesInput) (*service.GetMessagesOutput, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.GetMessagesOutput), args.Error(1)
}

func (m *MockSessionService) List(ctx context.Context, projectID uuid.UUID, spaceID *uuid.UUID, notConnected bool) ([]model.Session, error) {
	args := m.Called(ctx, projectID, spaceID, notConnected)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.Session), args.Error(1)
}

func setupSessionRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestSessionHandler_GetSessions(t *testing.T) {
	projectID := uuid.New()
	spaceID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		setup          func(*MockSessionService)
		expectedStatus int
	}{
		{
			name:        "successful sessions retrieval - all sessions",
			queryParams: "",
			setup: func(svc *MockSessionService) {
				expectedSessions := []model.Session{
					{
						ID:        uuid.New(),
						ProjectID: projectID,
						Configs:   datatypes.JSONMap{"temperature": 0.7},
					},
					{
						ID:        uuid.New(),
						ProjectID: projectID,
						Configs:   datatypes.JSONMap{"model": "gpt-4"},
					},
				}
				svc.On("List", mock.Anything, projectID, (*uuid.UUID)(nil), false).Return(expectedSessions, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "successful sessions retrieval - filter by space_id",
			queryParams: "?space_id=" + spaceID.String(),
			setup: func(svc *MockSessionService) {
				expectedSessions := []model.Session{
					{
						ID:        uuid.New(),
						ProjectID: projectID,
						SpaceID:   &spaceID,
						Configs:   datatypes.JSONMap{},
					},
				}
				svc.On("List", mock.Anything, projectID, &spaceID, false).Return(expectedSessions, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "successful sessions retrieval - not connected",
			queryParams: "?not_connected=true",
			setup: func(svc *MockSessionService) {
				expectedSessions := []model.Session{
					{
						ID:        uuid.New(),
						ProjectID: projectID,
						SpaceID:   nil,
						Configs:   datatypes.JSONMap{},
					},
				}
				svc.On("List", mock.Anything, projectID, (*uuid.UUID)(nil), true).Return(expectedSessions, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "empty sessions list",
			queryParams: "",
			setup: func(svc *MockSessionService) {
				svc.On("List", mock.Anything, projectID, (*uuid.UUID)(nil), false).Return([]model.Session{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "invalid space_id",
			queryParams: "?space_id=invalid-uuid",
			setup: func(svc *MockSessionService) {
				// No service call expected
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "service layer error",
			queryParams: "",
			setup: func(svc *MockSessionService) {
				svc.On("List", mock.Anything, projectID, (*uuid.UUID)(nil), false).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.GET("/session", func(c *gin.Context) {
				project := &model.Project{ID: projectID}
				c.Set("project", project)
				handler.GetSessions(c)
			})

			req := httptest.NewRequest("GET", "/session"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSessionHandler_CreateSession(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name           string
		requestBody    CreateSessionReq
		setup          func(*MockSessionService)
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful session creation",
			requestBody: CreateSessionReq{
				Configs: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens":  1000,
				},
			},
			setup: func(svc *MockSessionService) {
				svc.On("Create", mock.Anything, mock.MatchedBy(func(s *model.Session) bool {
					return s.ProjectID == projectID
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "session creation with space ID",
			requestBody: CreateSessionReq{
				SpaceID: uuid.New().String(),
				Configs: map[string]interface{}{
					"model": "gpt-4",
				},
			},
			setup: func(svc *MockSessionService) {
				svc.On("Create", mock.Anything, mock.MatchedBy(func(s *model.Session) bool {
					return s.ProjectID == projectID && s.SpaceID != nil
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedError:  false,
		},
		{
			name: "invalid space ID",
			requestBody: CreateSessionReq{
				SpaceID: "invalid-uuid",
				Configs: map[string]interface{}{},
			},
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "service layer error",
			requestBody: CreateSessionReq{
				Configs: map[string]interface{}{},
			},
			setup: func(svc *MockSessionService) {
				svc.On("Create", mock.Anything, mock.Anything).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.POST("/session", func(c *gin.Context) {
				// Simulate middleware setting project information
				project := &model.Project{ID: projectID}
				c.Set("project", project)
				handler.CreateSession(c)
			})

			body, _ := sonic.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/session", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSessionHandler_DeleteSession(t *testing.T) {
	projectID := uuid.New()
	sessionID := uuid.New()

	tests := []struct {
		name           string
		sessionIDParam string
		setup          func(*MockSessionService)
		expectedStatus int
	}{
		{
			name:           "successful session deletion",
			sessionIDParam: sessionID.String(),
			setup: func(svc *MockSessionService) {
				svc.On("Delete", mock.Anything, projectID, sessionID).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid session ID",
			sessionIDParam: "invalid-uuid",
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service layer error",
			sessionIDParam: sessionID.String(),
			setup: func(svc *MockSessionService) {
				svc.On("Delete", mock.Anything, projectID, sessionID).Return(errors.New("deletion failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.DELETE("/session/:session_id", func(c *gin.Context) {
				project := &model.Project{ID: projectID}
				c.Set("project", project)
				handler.DeleteSession(c)
			})

			req := httptest.NewRequest("DELETE", "/session/"+tt.sessionIDParam, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSessionHandler_UpdateConfigs(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name           string
		sessionIDParam string
		requestBody    UpdateSessionConfigsReq
		setup          func(*MockSessionService)
		expectedStatus int
	}{
		{
			name:           "successful config update",
			sessionIDParam: sessionID.String(),
			requestBody: UpdateSessionConfigsReq{
				Configs: map[string]interface{}{
					"temperature": 0.8,
					"max_tokens":  2000,
				},
			},
			setup: func(svc *MockSessionService) {
				svc.On("UpdateByID", mock.Anything, mock.MatchedBy(func(s *model.Session) bool {
					return s.ID == sessionID
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid session ID",
			sessionIDParam: "invalid-uuid",
			requestBody: UpdateSessionConfigsReq{
				Configs: map[string]interface{}{},
			},
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service layer error",
			sessionIDParam: sessionID.String(),
			requestBody: UpdateSessionConfigsReq{
				Configs: map[string]interface{}{},
			},
			setup: func(svc *MockSessionService) {
				svc.On("UpdateByID", mock.Anything, mock.Anything).Return(errors.New("update failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.PUT("/session/:session_id/configs", handler.UpdateConfigs)

			body, _ := sonic.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/session/"+tt.sessionIDParam+"/configs", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSessionHandler_GetConfigs(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name           string
		sessionIDParam string
		setup          func(*MockSessionService)
		expectedStatus int
	}{
		{
			name:           "successful config retrieval",
			sessionIDParam: sessionID.String(),
			setup: func(svc *MockSessionService) {
				expectedSession := &model.Session{
					ID:      sessionID,
					Configs: datatypes.JSONMap{"temperature": 0.7},
				}
				svc.On("GetByID", mock.Anything, mock.MatchedBy(func(s *model.Session) bool {
					return s.ID == sessionID
				})).Return(expectedSession, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid session ID",
			sessionIDParam: "invalid-uuid",
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service layer error",
			sessionIDParam: sessionID.String(),
			setup: func(svc *MockSessionService) {
				svc.On("GetByID", mock.Anything, mock.Anything).Return(nil, errors.New("session not found"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.GET("/session/:session_id/configs", handler.GetConfigs)

			req := httptest.NewRequest("GET", "/session/"+tt.sessionIDParam+"/configs", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSessionHandler_ConnectToSpace(t *testing.T) {
	sessionID := uuid.New()
	spaceID := uuid.New()

	tests := []struct {
		name           string
		sessionIDParam string
		requestBody    ConnectToSpaceReq
		setup          func(*MockSessionService)
		expectedStatus int
	}{
		{
			name:           "successful space connection",
			sessionIDParam: sessionID.String(),
			requestBody: ConnectToSpaceReq{
				SpaceID: spaceID.String(),
			},
			setup: func(svc *MockSessionService) {
				svc.On("UpdateByID", mock.Anything, mock.MatchedBy(func(s *model.Session) bool {
					return s.ID == sessionID && s.SpaceID != nil && *s.SpaceID == spaceID
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid session ID",
			sessionIDParam: "invalid-uuid",
			requestBody: ConnectToSpaceReq{
				SpaceID: spaceID.String(),
			},
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid space ID",
			sessionIDParam: sessionID.String(),
			requestBody: ConnectToSpaceReq{
				SpaceID: "invalid-uuid",
			},
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service layer error",
			sessionIDParam: sessionID.String(),
			requestBody: ConnectToSpaceReq{
				SpaceID: spaceID.String(),
			},
			setup: func(svc *MockSessionService) {
				svc.On("UpdateByID", mock.Anything, mock.Anything).Return(errors.New("connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.POST("/session/:session_id/connect_to_space", handler.ConnectToSpace)

			body, _ := sonic.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/session/"+tt.sessionIDParam+"/connect_to_space", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSessionHandler_SendMessage(t *testing.T) {
	projectID := uuid.New()
	sessionID := uuid.New()

	tests := []struct {
		name           string
		sessionIDParam string
		requestBody    SendMessageReq
		setup          func(*MockSessionService)
		expectedStatus int
	}{
		{
			name:           "successful text message sending",
			sessionIDParam: sessionID.String(),
			requestBody: SendMessageReq{
				Role: "user",
				Parts: []service.PartIn{
					{
						Type: "text",
						Text: "Hello, world!",
					},
				},
			},
			setup: func(svc *MockSessionService) {
				expectedMessage := &model.Message{
					ID:        uuid.New(),
					SessionID: sessionID,
					Role:      "user",
				}
				svc.On("SendMessage", mock.Anything, mock.MatchedBy(func(in service.SendMessageInput) bool {
					return in.ProjectID == projectID && in.SessionID == sessionID && in.Role == "user"
				})).Return(expectedMessage, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid session ID",
			sessionIDParam: "invalid-uuid",
			requestBody: SendMessageReq{
				Role: "user",
				Parts: []service.PartIn{
					{Type: "text", Text: "Hello"},
				},
			},
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid role",
			sessionIDParam: sessionID.String(),
			requestBody: SendMessageReq{
				Role: "invalid_role",
				Parts: []service.PartIn{
					{Type: "text", Text: "Hello"},
				},
			},
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service layer error",
			sessionIDParam: sessionID.String(),
			requestBody: SendMessageReq{
				Role: "user",
				Parts: []service.PartIn{
					{Type: "text", Text: "Hello"},
				},
			},
			setup: func(svc *MockSessionService) {
				svc.On("SendMessage", mock.Anything, mock.Anything).Return(nil, errors.New("send failed"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.POST("/session/:session_id/messages", func(c *gin.Context) {
				project := &model.Project{ID: projectID}
				c.Set("project", project)
				handler.SendMessage(c)
			})

			body, _ := sonic.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/session/"+tt.sessionIDParam+"/messages", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSessionHandler_GetMessages(t *testing.T) {
	sessionID := uuid.New()

	tests := []struct {
		name           string
		sessionIDParam string
		queryParams    string
		setup          func(*MockSessionService)
		expectedStatus int
	}{
		{
			name:           "successful message retrieval",
			sessionIDParam: sessionID.String(),
			queryParams:    "?limit=20",
			setup: func(svc *MockSessionService) {
				expectedOutput := &service.GetMessagesOutput{
					Items: []model.Message{
						{
							ID:        uuid.New(),
							SessionID: sessionID,
							Role:      "user",
						},
					},
					HasMore: false,
				}
				svc.On("GetMessages", mock.Anything, mock.MatchedBy(func(in service.GetMessagesInput) bool {
					return in.SessionID == sessionID && in.Limit == 20
				})).Return(expectedOutput, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid session ID",
			sessionIDParam: "invalid-uuid",
			queryParams:    "?limit=20",
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid limit parameter",
			sessionIDParam: sessionID.String(),
			queryParams:    "?limit=0",
			setup:          func(svc *MockSessionService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "service layer error",
			sessionIDParam: sessionID.String(),
			queryParams:    "?limit=20",
			setup: func(svc *MockSessionService) {
				svc.On("GetMessages", mock.Anything, mock.Anything).Return(nil, errors.New("retrieval failed"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockSessionService{}
			tt.setup(mockService)

			handler := NewSessionHandler(mockService)
			router := setupSessionRouter()
			router.GET("/session/:session_id/messages", handler.GetMessages)

			req := httptest.NewRequest("GET", "/session/"+tt.sessionIDParam+"/messages"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
