package path

import (
	"errors"
	"strings"
)

var (
	ErrEmptyPath     = errors.New("path cannot be empty")
	ErrInvalidPath   = errors.New("path format is invalid")
	ErrPathTraversal = errors.New("path contains directory traversal")
)

// ValidatePath validates a path for directory tree format
// It checks for:
// - Empty paths
// - Directory traversal attempts
// - Basic path format validation
func ValidatePath(path string) error {
	if path == "" {
		return ErrEmptyPath
	}

	// Check for directory traversal
	if strings.Contains(path, "..") {
		return ErrPathTraversal
	}

	// Allow root directory path "/"
	if path == "/" {
		return nil
	}

	// Split path into parts
	parts := strings.Split(path, "/")

	// Check each part
	for _, part := range parts {
		if part == "" {
			continue // Allow empty parts (for leading/trailing slashes)
		}

		// Check if part contains only dots (like "...")
		if strings.Trim(part, ".") == "" && len(part) > 1 {
			return ErrPathTraversal
		}

		// Check for null byte (security concern)
		if strings.Contains(part, "\x00") {
			return ErrInvalidPath
		}
	}

	return nil
}

// SanitizePath cleans and sanitizes a path, removing potentially dangerous elements
func SanitizePath(path string) string {
	// Remove leading slashes
	cleanPath := strings.TrimPrefix(path, "/")

	// Remove leading dots
	cleanPath = strings.TrimPrefix(cleanPath, "./")

	// Replace null bytes with underscores (security concern)
	cleanPath = strings.ReplaceAll(cleanPath, "\x00", "_")

	// Ensure the path doesn't start with dots
	if strings.HasPrefix(cleanPath, ".") {
		cleanPath = "file_" + cleanPath
	}

	return cleanPath
}

// GetDirectoriesFromPaths extracts unique directory names from a list of file paths
// that are direct children of the given parent path
func GetDirectoriesFromPaths(parentPath string, filePaths []string) []string {
	if parentPath == "" {
		parentPath = "/"
	}

	// Normalize parent path - ensure it starts with / and ends with / (except for root)
	parentPath = strings.TrimSpace(parentPath)
	if !strings.HasPrefix(parentPath, "/") {
		parentPath = "/" + parentPath
	}
	if parentPath != "/" && !strings.HasSuffix(parentPath, "/") {
		parentPath = parentPath + "/"
	}

	directories := make(map[string]bool)

	for _, filePath := range filePaths {
		// Normalize file path
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}

		// Ensure filePath starts with /
		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
		}

		// Skip if path doesn't start with parent path
		if !strings.HasPrefix(filePath, parentPath) {
			continue
		}

		// Get the relative path from parent
		relativePath := strings.TrimPrefix(filePath, parentPath)

		// Skip empty relative path (file directly in parent path)
		if relativePath == "" {
			continue
		}

		// Split by / and get the first part (direct child)
		parts := strings.Split(relativePath, "/")
		if len(parts) > 0 && parts[0] != "" {
			// This is a direct child directory
			directories[parts[0]] = true
		}
	}

	// Convert map keys to slice
	result := make([]string, 0, len(directories))
	for dir := range directories {
		result = append(result, dir)
	}

	return result
}
