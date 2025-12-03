package models

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.uber.org/zap"
	"lloader/internal/app"
)

type Model struct {
	Name string
	Path string
	Size int64
}

func DiscoverModels(cfg *app.Config, logger *zap.Logger) ([]Model, error) {
	logger.Info("Discovering models", zap.String("directory", cfg.ModelsDir))

	entries, err := os.ReadDir(cfg.ModelsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read models directory: %w", err)
	}

	var models []Model
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isModelFile(name) {
			continue
		}

		path := filepath.Join(cfg.ModelsDir, name)
		info, err := entry.Info()
		if err != nil {
			logger.Warn("Failed to get file info", zap.String("file", name), zap.Error(err))
			continue
		}

		models = append(models, Model{
			Name: name,
			Path: path,
			Size: info.Size(),
		})

		logger.Debug("Found model", zap.String("name", name), zap.Int64("size", info.Size()))
	}

	logger.Info("Discovered models", zap.Int("count", len(models)))
	return models, nil
}

func isModelFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	modelExtensions := []string{".gguf", ".ggml", ".bin", ".model"}

	return slices.Contains(modelExtensions, ext)
}

func GetModelNames(models []Model) []string {
	names := make([]string, len(models))
	for i, model := range models {
		names[i] = model.Name
	}
	return names
}
