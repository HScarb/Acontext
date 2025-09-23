package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/memodb-io/Acontext/internal/modules/model"
	"github.com/memodb-io/Acontext/internal/modules/serializer"
	"github.com/memodb-io/Acontext/internal/modules/service"
	"github.com/memodb-io/Acontext/internal/pkg/utils/path"
)

type ArtifactHandler struct {
	svc service.ArtifactService
}

func NewArtifactHandler(s service.ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{svc: s}
}

type CreateArtifactReq struct {
	Path string `form:"path" json:"path"` // Optional, defaults to "/" (root directory)
	Meta string `form:"meta" json:"meta"`
}

// CreateArtifact godoc
//
//	@Summary		Create artifact
//	@Description	Upload a file and create an artifact under a project
//	@Tags			artifact
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			path	formData	string	false	"File path in the artifact storage (optional, defaults to root '/')"
//	@Param			file	formData	file	true	"File to upload"
//	@Param			meta	formData	string	false	"Custom metadata as JSON string (optional, system metadata will be stored under '_system' key)"
//	@Security		BearerAuth
//	@Success		201	{object}	serializer.Response{data=model.Artifact}
//	@Router			/artifact [post]
func (h *ArtifactHandler) CreateArtifact(c *gin.Context) {
	req := CreateArtifactReq{}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", err))
		return
	}

	// Set default path to root directory if not provided
	if req.Path == "" {
		req.Path = "/"
	}

	// Validate the path parameter
	if err := path.ValidatePath(req.Path); err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("invalid path", err))
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("file is required", err))
		return
	}

	project, ok := c.MustGet("project").(*model.Project)
	if !ok {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", errors.New("project not found")))
		return
	}

	// Parse user meta from JSON string
	var userMeta map[string]interface{}
	if req.Meta != "" {
		if err := json.Unmarshal([]byte(req.Meta), &userMeta); err != nil {
			c.JSON(http.StatusBadRequest, serializer.ParamErr("invalid meta JSON format", err))
			return
		}

		// Validate that user meta doesn't contain system reserved keys
		reservedKeys := []string{"_system"}
		for _, reservedKey := range reservedKeys {
			if _, exists := userMeta[reservedKey]; exists {
				c.JSON(http.StatusBadRequest, serializer.ParamErr("", fmt.Errorf("reserved key '%s' is not allowed in user meta", reservedKey)))
				return
			}
		}
	}

	artifact, err := h.svc.Create(c.Request.Context(), project.ID, req.Path, file, userMeta)
	if err != nil {
		c.JSON(http.StatusInternalServerError, serializer.DBErr("", err))
		return
	}

	c.JSON(http.StatusCreated, serializer.Response{Data: artifact})
}

// DeleteArtifact godoc
//
//	@Summary		Delete artifact
//	@Description	Delete an artifact by its UUID
//	@Tags			artifact
//	@Accept			json
//	@Produce		json
//	@Param			artifact_id	path	string	true	"Artifact ID"	Format(uuid)	Example(123e4567-e89b-12d3-a456-426614174000)
//	@Security		BearerAuth
//	@Success		200	{object}	serializer.Response{}
//	@Router			/artifact/{artifact_id} [delete]
func (h *ArtifactHandler) DeleteArtifact(c *gin.Context) {
	artifactID, err := uuid.Parse(c.Param("artifact_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", err))
		return
	}

	project, ok := c.MustGet("project").(*model.Project)
	if !ok {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", errors.New("project not found")))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), project.ID, artifactID); err != nil {
		c.JSON(http.StatusInternalServerError, serializer.DBErr("", err))
		return
	}

	c.JSON(http.StatusOK, serializer.Response{})
}

type GetArtifactReq struct {
	WithPublicURL bool `form:"with_public_url,default=false" json:"with_public_url" example:"false"`
	Expire        int  `form:"expire,default=3600" json:"expire" example:"3600"` // Expire time in seconds for presigned URL
}

type GetArtifactResp struct {
	Artifact  *model.Artifact `json:"artifact"`
	PublicURL *string         `json:"public_url,omitempty"`
}

// GetArtifact godoc
//
//	@Summary		Get artifact
//	@Description	Get artifact information by its UUID. Optionally include a presigned URL for downloading.
//	@Tags			artifact
//	@Accept			json
//	@Produce		json
//	@Param			artifact_id		path	string	true	"Artifact ID"												Format(uuid)	Example(123e4567-e89b-12d3-a456-426614174000)
//	@Param			with_public_url	query	boolean	false	"Whether to return public URL, default is false"			example:"false"
//	@Param			expire			query	int		false	"Expire time in seconds for presigned URL (default: 3600)"	example:"3600"
//	@Security		BearerAuth
//	@Success		200	{object}	serializer.Response{data=handler.GetArtifactResp}
//	@Router			/artifact/{artifact_id} [get]
func (h *ArtifactHandler) GetArtifact(c *gin.Context) {
	req := GetArtifactReq{}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", err))
		return
	}

	artifactID, err := uuid.Parse(c.Param("artifact_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", err))
		return
	}

	project, ok := c.MustGet("project").(*model.Project)
	if !ok {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", errors.New("project not found")))
		return
	}

	artifact, err := h.svc.GetByID(c.Request.Context(), project.ID, artifactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, serializer.DBErr("", err))
		return
	}

	resp := GetArtifactResp{Artifact: artifact}

	// Generate presigned URL if requested
	if req.WithPublicURL {
		url, err := h.svc.GetPresignedURL(c.Request.Context(), project.ID, artifactID, time.Duration(req.Expire)*time.Second)
		if err != nil {
			c.JSON(http.StatusInternalServerError, serializer.DBErr("", err))
			return
		}
		resp.PublicURL = &url
	}

	c.JSON(http.StatusOK, serializer.Response{Data: resp})
}

type UpdateArtifactReq struct {
	Path string `form:"path" json:"path"` // Optional new path
}

type UpdateArtifactResp struct {
	Artifact *model.Artifact `json:"artifact"`
}

// UpdateArtifact godoc
//
//	@Summary		Update artifact
//	@Description	Update an artifact by uploading a new file
//	@Tags			artifact
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			artifact_id	path		string	true	"Artifact ID"	Format(uuid)	Example(123e4567-e89b-12d3-a456-426614174000)
//	@Param			file		formData	file	true	"New file to upload"
//	@Param			path		formData	string	false	"New file path (optional)"
//	@Security		BearerAuth
//	@Success		200	{object}	serializer.Response{data=handler.UpdateArtifactResp}
//	@Router			/artifact/{artifact_id} [put]
func (h *ArtifactHandler) UpdateArtifact(c *gin.Context) {
	req := UpdateArtifactReq{}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", err))
		return
	}

	// Validate the path parameter if provided
	if req.Path != "" {
		if err := path.ValidatePath(req.Path); err != nil {
			c.JSON(http.StatusBadRequest, serializer.ParamErr("invalid path", err))
			return
		}
	}

	artifactID, err := uuid.Parse(c.Param("artifact_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", err))
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("file is required", err))
		return
	}

	project, ok := c.MustGet("project").(*model.Project)
	if !ok {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", errors.New("project not found")))
		return
	}

	artifact, err := h.svc.UpdateFile(c.Request.Context(), project.ID, artifactID, file, &req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, serializer.DBErr("", err))
		return
	}

	c.JSON(http.StatusOK, serializer.Response{
		Data: UpdateArtifactResp{Artifact: artifact},
	})
}

type ListArtifactsReq struct {
	Path string `form:"path" json:"path"` // Optional path filter
}

type ListArtifactsResp struct {
	Artifacts   []*model.Artifact `json:"artifacts"`
	Directories []string          `json:"directories"`
}

// ListArtifacts godoc
//
//	@Summary		List artifacts
//	@Description	List artifacts in a specific path or all artifacts
//	@Tags			artifact
//	@Accept			json
//	@Produce		json
//	@Param			path	query	string	false	"Path filter (optional, defaults to root '/')"
//	@Security		BearerAuth
//	@Success		200	{object}	serializer.Response{data=handler.ListArtifactsResp}
//	@Router			/artifact [get]
func (h *ArtifactHandler) ListArtifacts(c *gin.Context) {
	project, ok := c.MustGet("project").(*model.Project)
	if !ok {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("", errors.New("project not found")))
		return
	}

	pathQuery := c.Query("path")

	// Set default path to root directory if not provided
	if pathQuery == "" {
		pathQuery = "/"
	}

	// Validate the path parameter
	if err := path.ValidatePath(pathQuery); err != nil {
		c.JSON(http.StatusBadRequest, serializer.ParamErr("invalid path", err))
		return
	}

	artifacts, err := h.svc.ListByPath(c.Request.Context(), project.ID, pathQuery)
	if err != nil {
		c.JSON(http.StatusInternalServerError, serializer.DBErr("", err))
		return
	}

	// Get all paths to extract directory names
	allPaths, err := h.svc.GetAllPaths(c.Request.Context(), project.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, serializer.DBErr("", err))
		return
	}

	// Extract direct subdirectories
	directories := path.GetDirectoriesFromPaths(pathQuery, allPaths)

	c.JSON(http.StatusOK, serializer.Response{
		Data: ListArtifactsResp{
			Artifacts:   artifacts,
			Directories: directories,
		},
	})
}
