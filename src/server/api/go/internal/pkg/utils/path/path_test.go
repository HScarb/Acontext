package path

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		errorType   error
	}{
		// Valid paths (directories only, no files)
		{
			name:        "valid simple directory",
			path:        "documents",
			expectError: false,
		},
		{
			name:        "valid nested directory",
			path:        "folder/subfolder",
			expectError: false,
		},
		{
			name:        "valid directory with special characters",
			path:        "folder/subfolder_v2.1",
			expectError: false,
		},
		{
			name:        "valid directory with numbers",
			path:        "2023/12/25",
			expectError: false,
		},
		{
			name:        "valid root directory path",
			path:        "/",
			expectError: false,
		},
		{
			name:        "valid path with leading slash",
			path:        "/folder/subfolder",
			expectError: false,
		},
		{
			name:        "valid path with trailing slash",
			path:        "folder/subfolder/",
			expectError: false,
		},
		{
			name:        "valid directory with special characters",
			path:        "folder<name>",
			expectError: false,
		},
		{
			name:        "valid directory with reserved name",
			path:        "CON",
			expectError: false,
		},
		{
			name:        "valid path with file extension (now allowed)",
			path:        "document.txt",
			expectError: false,
		},
		{
			name:        "valid nested path with file extension (now allowed)",
			path:        "folder/document.pdf",
			expectError: false,
		},

		// Empty path
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorType:   ErrEmptyPath,
		},

		// Directory traversal
		{
			name:        "directory traversal with ..",
			path:        "../secret.txt",
			expectError: true,
			errorType:   ErrPathTraversal,
		},
		{
			name:        "directory traversal in middle",
			path:        "folder/../secret.txt",
			expectError: true,
			errorType:   ErrPathTraversal,
		},
		{
			name:        "multiple dots",
			path:        ".../file.txt",
			expectError: true,
			errorType:   ErrPathTraversal,
		},
		{
			name:        "path with null byte",
			path:        "file\x00name.txt",
			expectError: true,
			errorType:   ErrInvalidPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean directory path",
			input:    "documents",
			expected: "documents",
		},
		{
			name:     "path with leading slash",
			input:    "/documents",
			expected: "documents",
		},
		{
			name:     "path with leading dots",
			input:    "./documents",
			expected: "documents",
		},
		{
			name:     "path with null byte",
			input:    "file\x00name.txt",
			expected: "file_name.txt",
		},
		{
			name:     "path starting with dots",
			input:    ".../secret",
			expected: "file_.../secret",
		},
		{
			name:     "path with special characters",
			input:    "file<name>",
			expected: "file<name>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDirectoriesFromPaths(t *testing.T) {
	tests := []struct {
		name       string
		parentPath string
		filePaths  []string
		expected   []string
	}{
		{
			name:       "root path with nested directories",
			parentPath: "/",
			filePaths: []string{
				"/documents/file1.txt",
				"/documents/file2.pdf",
				"/images/photo1.jpg",
				"/images/photo2.png",
				"/code/script.py",
			},
			expected: []string{"documents", "images", "code"},
		},
		{
			name:       "nested parent path",
			parentPath: "/documents",
			filePaths: []string{
				"/documents/work/project1.txt",
				"/documents/work/project2.txt",
				"/documents/personal/note1.txt",
				"/documents/personal/note2.txt",
				"/images/photo.jpg", // This should be ignored
			},
			expected: []string{"work", "personal"},
		},
		{
			name:       "parent path with trailing slash",
			parentPath: "/documents/",
			filePaths: []string{
				"/documents/work/project1.txt",
				"/documents/personal/note1.txt",
			},
			expected: []string{"work", "personal"},
		},
		{
			name:       "no matching paths",
			parentPath: "/nonexistent",
			filePaths: []string{
				"/documents/file1.txt",
				"/images/photo.jpg",
			},
			expected: []string{},
		},
		{
			name:       "files directly in parent path",
			parentPath: "/documents",
			filePaths: []string{
				"/documents/file1.txt",
				"/documents/file2.pdf",
			},
			expected: []string{"file1.txt", "file2.pdf"},
		},
		{
			name:       "empty parent path defaults to root",
			parentPath: "",
			filePaths: []string{
				"/documents/file1.txt",
				"/images/photo.jpg",
			},
			expected: []string{"documents", "images"},
		},
		{
			name:       "single directory",
			parentPath: "/",
			filePaths: []string{
				"/single/file.txt",
			},
			expected: []string{"single"},
		},
		{
			name:       "duplicate directories should be unique",
			parentPath: "/",
			filePaths: []string{
				"/documents/file1.txt",
				"/documents/file2.txt",
				"/images/photo1.jpg",
				"/images/photo2.jpg",
			},
			expected: []string{"documents", "images"},
		},
		{
			name:       "paths without leading slash should be normalized",
			parentPath: "/",
			filePaths: []string{
				"webp/image1.webp",
				"webp/image2.webp",
				"documents/file1.txt",
				"images/photo.jpg",
			},
			expected: []string{"webp", "documents", "images"},
		},
		{
			name:       "parent path without leading slash should be normalized",
			parentPath: "documents",
			filePaths: []string{
				"/documents/work/project1.txt",
				"/documents/personal/note1.txt",
				"/images/photo.jpg",
			},
			expected: []string{"work", "personal"},
		},
		{
			name:       "paths with extra spaces should be normalized",
			parentPath: " / ",
			filePaths: []string{
				" /webp/image1.webp ",
				"/documents/file1.txt",
				" /images/photo.jpg ",
			},
			expected: []string{"webp", "documents", "images"},
		},
		{
			name:       "root path query with single directory",
			parentPath: "/",
			filePaths: []string{
				"/",
				"/webp",
			},
			expected: []string{"webp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDirectoriesFromPaths(tt.parentPath, tt.filePaths)

			// Sort both slices for comparison since order doesn't matter
			sort.Strings(result)
			sort.Strings(tt.expected)

			assert.Equal(t, tt.expected, result)
		})
	}
}
