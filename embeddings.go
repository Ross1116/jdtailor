package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type semanticEmbedding struct {
	EntityType string
	EntityID   int64
	Provider   string
	Model      string
	InputHash  string
	Dimensions int
	Vector     []float64
}

func (s *Store) embeddingForEntity(ctx context.Context, client *http.Client, entityType string, entityID int64, text string) ([]float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, errors.New("embedding text is empty")
	}
	settings, err := s.GetSettings()
	if err != nil {
		return nil, err
	}
	provider := configuredProvider(settings.Provider)
	model := configuredEmbeddingModel(provider, settings.EmbeddingModel)
	inputHash := embeddingInputHash(provider, model, text)
	if cached, err := s.getCachedEmbedding(entityType, entityID, provider, model, inputHash); err == nil && len(cached.Vector) > 0 {
		return cached.Vector, nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		_ = s.LogEvent("warning", "embedding cache read failed, regenerating: "+err.Error())
	}

	vector, provider, model, err := s.GenerateEmbedding(ctx, client, text)
	if err != nil {
		return nil, err
	}
	inputHash = embeddingInputHash(provider, model, text)
	if err := s.saveCachedEmbedding(semanticEmbedding{
		EntityType: entityType,
		EntityID:   entityID,
		Provider:   provider,
		Model:      model,
		InputHash:  inputHash,
		Dimensions: len(vector),
		Vector:     vector,
	}); err != nil {
		return nil, err
	}
	return vector, nil
}

func (s *Store) getCachedEmbedding(entityType string, entityID int64, provider string, model string, inputHash string) (semanticEmbedding, error) {
	var cached semanticEmbedding
	var vectorJSON string
	err := s.db.QueryRowContext(
		context.Background(),
		`SELECT entity_type, entity_id, provider, model, input_hash, dimensions, vector_json
		FROM semantic_embeddings
		WHERE entity_type = ? AND entity_id = ? AND provider = ? AND model = ? AND input_hash = ?`,
		entityType,
		entityID,
		provider,
		model,
		inputHash,
	).Scan(&cached.EntityType, &cached.EntityID, &cached.Provider, &cached.Model, &cached.InputHash, &cached.Dimensions, &vectorJSON)
	if err != nil {
		return semanticEmbedding{}, err
	}
	if err := json.Unmarshal([]byte(vectorJSON), &cached.Vector); err != nil {
		return semanticEmbedding{}, err
	}
	return cached, nil
}

func (s *Store) saveCachedEmbedding(embedding semanticEmbedding) error {
	vectorJSON, err := json.Marshal(embedding.Vector)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(
		context.Background(),
		`INSERT INTO semantic_embeddings
			(entity_type, entity_id, provider, model, input_hash, dimensions, vector_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(entity_type, entity_id, provider, model, input_hash)
		DO UPDATE SET dimensions = excluded.dimensions, vector_json = excluded.vector_json, updated_at = excluded.updated_at`,
		embedding.EntityType,
		embedding.EntityID,
		embedding.Provider,
		embedding.Model,
		embedding.InputHash,
		embedding.Dimensions,
		string(vectorJSON),
		now,
		now,
	)
	return err
}

func cosineSimilarity(left []float64, right []float64) float64 {
	if len(left) == 0 || len(right) == 0 || len(left) != len(right) {
		return 0
	}
	dot := 0.0
	leftNorm := 0.0
	rightNorm := 0.0
	for index := range left {
		dot += left[index] * right[index]
		leftNorm += left[index] * left[index]
		rightNorm += right[index] * right[index]
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (sqrtFloat(leftNorm) * sqrtFloat(rightNorm))
}

func sqrtFloat(value float64) float64 {
	if value <= 0 {
		return 0
	}
	x := value
	for i := 0; i < 12; i++ {
		x = 0.5 * (x + value/x)
	}
	return x
}
