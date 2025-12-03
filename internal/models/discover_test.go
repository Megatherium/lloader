package models

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"lloader/internal/app"
)

func TestIsModelFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"GGUF file", "model.gguf", true},
		{"GGML file", "model.ggml", true},
		{"BIN file", "model.bin", true},
		{"MODEL file", "model.model", true},
		{"Text file", "model.txt", false},
		{"No extension", "model", false},
		{"Capital extension", "model.GGUF", true},
		{"Mixed case", "model.GgUf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isModelFile(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetModelNames(t *testing.T) {
	models := []Model{
		{Name: "model1.gguf", Path: "/path/to/model1.gguf", Size: 1000},
		{Name: "model2.ggml", Path: "/path/to/model2.ggml", Size: 2000},
		{Name: "model3.bin", Path: "/path/to/model3.bin", Size: 3000},
	}

	names := GetModelNames(models)
	expected := []string{"model1.gguf", "model2.ggml", "model3.bin"}

	assert.Equal(t, expected, names)
	assert.Len(t, names, 3)
}

func TestDiscoverModels_EmptyDir(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	cfg := &app.Config{
		ModelsDir: tempDir,
	}

	logger := zap.NewNop()
	models, err := DiscoverModels(cfg, logger)

	assert.NoError(t, err)
	assert.Empty(t, models)
}

func TestDiscoverModels_WithFiles(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test files
	files := []struct {
		name    string
		content string
		isModel bool
	}{
		{"test.gguf", "gguf content", true},
		{"test.txt", "text content", false},
		{"another.ggml", "ggml content", true},
		{"data.bin", "bin content", true},
	}

	for _, file := range files {
		path := filepath.Join(tempDir, file.name)
		err := os.WriteFile(path, []byte(file.content), 0644)
		assert.NoError(t, err)
	}

	cfg := &app.Config{
		ModelsDir: tempDir,
	}

	logger := zap.NewNop()
	models, err := DiscoverModels(cfg, logger)

	assert.NoError(t, err)
	assert.Len(t, models, 3) // Should find 3 model files

	// Check that we found the right files
	foundNames := make(map[string]bool)
	for _, model := range models {
		foundNames[model.Name] = true
	}

	assert.True(t, foundNames["test.gguf"])
	assert.True(t, foundNames["another.ggml"])
	assert.True(t, foundNames["data.bin"])
	assert.False(t, foundNames["test.txt"])
}
