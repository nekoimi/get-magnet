package ai_repo

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/nekoimi/get-magnet/internal/ai"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/db/table"
)

func GetByCacheKey(key string) (*table.AIExtraction, bool, error) {
	if db.Instance() == nil {
		return nil, false, errors.New("database is not initialized")
	}
	row := new(table.AIExtraction)
	has, err := db.Instance().Where("cache_key = ?", key).Get(row)
	return row, has, err
}

func Save(cacheKey string, request ai.Request, result ai.Result, documentID, versionID *int64) (*table.AIExtraction, error) {
	if db.Instance() == nil {
		return nil, errors.New("database is not initialized")
	}
	encoded, err := json.Marshal(result.Values)
	if err != nil {
		return nil, err
	}
	row := &table.AIExtraction{CacheKey: cacheKey, DocumentId: documentID, WorkflowVersionId: versionID, Model: result.Model, Result: string(encoded), Confidence: result.Confidence, ReviewStatus: result.ReviewStatus, InputTokens: result.InputTokens, OutputTokens: result.OutputTokens, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := db.Instance().InsertOne(row); err != nil {
		existing, has, getErr := GetByCacheKey(cacheKey)
		if getErr == nil && has {
			return existing, nil
		}
		return nil, err
	}
	return row, nil
}
