// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package execution

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LerianStudio/flowker/internal/services/command"
	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	nethttp "github.com/LerianStudio/flowker/pkg/net/http"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// CollectionName is the MongoDB collection name for workflow executions.
	CollectionName = "workflow_executions"
)

// MongoDBRepository implements command.ExecutionRepository using MongoDB.
type MongoDBRepository struct {
	collection *mongo.Collection
}

// NewMongoDBRepository creates a new MongoDB repository for workflow executions.
func NewMongoDBRepository(db *mongo.Database) *MongoDBRepository {
	return &MongoDBRepository{
		collection: db.Collection(CollectionName),
	}
}

// Verify MongoDBRepository implements command.ExecutionRepository
var _ command.ExecutionRepository = (*MongoDBRepository)(nil)

// Create persists a new workflow execution to MongoDB.
func (r *MongoDBRepository) Create(ctx context.Context, execution *model.WorkflowExecution) error {
	mongoModel := NewMongoDBModelFromEntity(execution)

	_, err := r.collection.InsertOne(ctx, mongoModel)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return constant.ErrExecutionDuplicate
		}

		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// FindByID retrieves a workflow execution by its ID.
func (r *MongoDBRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	filter := bson.M{"executionId": id.String()}

	var mongoModel MongoDBModel

	err := r.collection.FindOne(ctx, filter).Decode(&mongoModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, constant.ErrExecutionNotFound
		}

		return nil, fmt.Errorf("failed to find execution by ID: %w", err)
	}

	entity, toErr := mongoModel.ToEntity()
	if toErr != nil {
		return nil, fmt.Errorf("failed to convert execution entity: %w", toErr)
	}

	return entity, nil
}

// FindByIdempotencyKey retrieves a workflow execution by its idempotency key.
func (r *MongoDBRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.WorkflowExecution, error) {
	filter := bson.M{"idempotencyKey": key}

	var mongoModel MongoDBModel

	err := r.collection.FindOne(ctx, filter).Decode(&mongoModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, constant.ErrExecutionNotFound
		}

		return nil, fmt.Errorf("failed to find execution by idempotency key: %w", err)
	}

	entity, toErr := mongoModel.ToEntity()
	if toErr != nil {
		return nil, fmt.Errorf("failed to convert execution entity: %w", toErr)
	}

	return entity, nil
}

// Update persists changes to an existing workflow execution.
// expectedStatus is included in the filter to prevent race conditions (atomic check-and-set).
func (r *MongoDBRepository) Update(ctx context.Context, execution *model.WorkflowExecution, expectedStatus model.ExecutionStatus) error {
	filter := bson.M{"executionId": execution.ExecutionID().String()}
	if expectedStatus != "" {
		filter["status"] = string(expectedStatus)
	}

	mongoModel := NewMongoDBModelFromEntity(execution)

	update := bson.M{
		"$set": bson.M{
			"status":            mongoModel.Status,
			"outputData":        mongoModel.OutputData,
			"errorMessage":      mongoModel.ErrorMessage,
			"currentStepNumber": mongoModel.CurrentStepNumber,
			"steps":             mongoModel.Steps,
			"completedAt":       mongoModel.CompletedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	if result.MatchedCount == 0 {
		if expectedStatus != "" {
			return constant.ErrConflictStateChanged
		}

		return constant.ErrExecutionNotFound
	}

	return nil
}

// FindIncomplete retrieves all executions with status pending or running.
func (r *MongoDBRepository) FindIncomplete(ctx context.Context) ([]*model.WorkflowExecution, error) {
	filter := bson.M{
		"status": bson.M{
			"$in": []string{
				string(model.ExecutionStatusPending),
				string(model.ExecutionStatusRunning),
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find incomplete executions: %w", err)
	}
	defer cursor.Close(ctx)

	var mongoModels []MongoDBModel
	if err := cursor.All(ctx, &mongoModels); err != nil {
		return nil, fmt.Errorf("failed to decode incomplete executions: %w", err)
	}

	executions := make([]*model.WorkflowExecution, len(mongoModels))

	for i, m := range mongoModels {
		entity, toErr := m.ToEntity()
		if toErr != nil {
			return nil, fmt.Errorf("failed to convert execution at index %d: %w", i, toErr)
		}

		executions[i] = entity
	}

	return executions, nil
}

// List retrieves executions with filtering and cursor pagination.
func (r *MongoDBRepository) List(ctx context.Context, filter command.ExecutionListFilter) (*command.ExecutionListResult, error) {
	mongoFilter := bson.M{}

	if filter.WorkflowID != nil {
		mongoFilter["workflowId"] = filter.WorkflowID.String()
	}

	if filter.Status != nil {
		mongoFilter["status"] = string(*filter.Status)
	}

	sortBy := filter.SortBy
	sortOrder := filter.SortOrder

	if filter.Cursor != "" {
		var err error

		sortBy, sortOrder, err = r.applyCursorFilter(filter.Cursor, sortBy, sortOrder, mongoFilter)
		if err != nil {
			return nil, err
		}
	}

	// Ensure canonical values so queries and encoded cursors stay consistent.
	sortBy, sortOrder = normalizeSortParams(sortBy, sortOrder)

	limit := normalizeLimit(filter.Limit)
	findOptions := options.Find().
		SetSort(buildSortOpts(sortBy, sortOrder)).
		SetLimit(int64(limit + 1))

	cursor, err := r.collection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer cursor.Close(ctx)

	var mongoModels []MongoDBModel
	if err := cursor.All(ctx, &mongoModels); err != nil {
		return nil, fmt.Errorf("failed to decode executions: %w", err)
	}

	hasMore := len(mongoModels) > limit
	if hasMore {
		mongoModels = mongoModels[:limit]
	}

	items, err := r.decodeExecutions(mongoModels)
	if err != nil {
		return nil, err
	}

	nextCursor, err := buildNextCursor(items, hasMore, sortBy, sortOrder)
	if err != nil {
		return nil, err
	}

	return &command.ExecutionListResult{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// applyCursorFilter decodes the cursor, validates its sort parameters, and applies
// the keyset pagination condition to the MongoDB filter.
func (r *MongoDBRepository) applyCursorFilter(cursorStr, sortBy, sortOrder string, mongoFilter bson.M) (string, string, error) {
	cur, err := nethttp.DecodeCursor(cursorStr)
	if err != nil {
		return "", "", pkg.ValidationError{Code: "INVALID_CURSOR", Message: "invalid cursor format", Err: err}
	}

	if cur.SortBy != "" {
		if !command.IsValidExecutionSortField(cur.SortBy) {
			return "", "", pkg.ValidationError{Code: "INVALID_CURSOR", Message: "invalid cursor sortBy"}
		}

		sortBy = cur.SortBy
	}

	if cur.SortOrder != "" {
		normalized := strings.ToUpper(cur.SortOrder)
		if normalized != "ASC" && normalized != "DESC" {
			return "", "", pkg.ValidationError{Code: "INVALID_CURSOR", Message: "invalid cursor sortOrder"}
		}

		sortOrder = normalized
	}

	sortField := mapExecutionSortField(sortBy)

	op := "$lt"
	if sortOrder == "ASC" {
		op = "$gt"
	}

	// Parse the cursor's sort value back to time.Time for correct BSON Date comparison.
	// The cursor stores timestamps as formatted strings, but MongoDB stores them as
	// native BSON Date. Comparing a BSON Date against a string yields zero results.
	sortValue, err := parseSortValue(cur.SortValue, sortBy)
	if err != nil {
		return "", "", pkg.ValidationError{Code: "INVALID_CURSOR", Message: "invalid cursor sort value", Err: err}
	}

	mongoFilter["$or"] = []bson.M{
		{sortField: bson.M{op: sortValue}},
		{sortField: sortValue, "executionId": bson.M{op: cur.ID}},
	}

	return sortBy, sortOrder, nil
}

// normalizeSortParams ensures sortBy and sortOrder have canonical values
// so that queries and encoded cursors remain consistent.
func normalizeSortParams(sortBy, sortOrder string) (string, string) {
	if !command.IsValidExecutionSortField(sortBy) {
		sortBy = command.DefaultExecutionSortField
	}

	sortOrder = strings.ToUpper(sortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	return sortBy, sortOrder
}

// normalizeLimit clamps limit to valid bounds.
func normalizeLimit(limit int) int {
	if limit <= 0 {
		return constant.DefaultPaginationLimit
	}

	if limit > constant.MaxPaginationLimit {
		return constant.MaxPaginationLimit
	}

	return limit
}

// buildSortOpts creates MongoDB sort options from sort parameters.
func buildSortOpts(sortBy, sortOrder string) bson.D {
	sortDirection := -1
	if sortOrder == "ASC" {
		sortDirection = 1
	}

	sortField := mapExecutionSortField(sortBy)

	return bson.D{
		{Key: sortField, Value: sortDirection},
		{Key: "executionId", Value: sortDirection},
	}
}

// decodeExecutions converts MongoDB models to domain entities.
func (r *MongoDBRepository) decodeExecutions(mongoModels []MongoDBModel) ([]*model.WorkflowExecution, error) {
	items := make([]*model.WorkflowExecution, len(mongoModels))

	for i, m := range mongoModels {
		entity, err := m.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert execution at index %d: %w", i, err)
		}

		items[i] = entity
	}

	return items, nil
}

// buildNextCursor encodes the pagination cursor from the last item if more results exist.
func buildNextCursor(items []*model.WorkflowExecution, hasMore bool, sortBy, sortOrder string) (string, error) {
	if !hasMore || len(items) == 0 {
		return "", nil
	}

	last := items[len(items)-1]

	encoded, err := nethttp.EncodeCursor(nethttp.Cursor{
		ID:         last.ExecutionID().String(),
		SortValue:  getExecutionSortValue(last, sortBy),
		SortBy:     sortBy,
		SortOrder:  sortOrder,
		PointsNext: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode cursor: %w", err)
	}

	return encoded, nil
}

// mapExecutionSortField maps API field names to MongoDB field names for executions.
func mapExecutionSortField(field string) string {
	switch field {
	case "startedAt":
		return "startedAt"
	case "completedAt":
		return "completedAt"
	default:
		return "startedAt"
	}
}

// parseSortValue converts a cursor's string sort value back to the correct Go type
// for MongoDB comparison. Time-based sort fields are parsed to time.Time so they
// match the BSON Date type stored in MongoDB.
func parseSortValue(value, sortBy string) (any, error) {
	switch sortBy {
	case "startedAt":
		if value == "" {
			return nil, fmt.Errorf("invalid empty time value in cursor for startedAt")
		}

		t, err := time.Parse(cursorTimeFormat, value)
		if err != nil {
			return nil, fmt.Errorf("invalid time value in cursor: %w", err)
		}

		return t, nil
	case "completedAt":
		if value == "" {
			// completedAt is nil for in-progress executions. Sorting by completedAt
			// with in-progress executions as the last item on a page is not supported —
			// the zero time filter (year 0001) would return no results on the next page.
			// Callers should filter to completed/failed executions when sorting by completedAt.
			return time.Time{}, nil
		}

		t, err := time.Parse(cursorTimeFormat, value)
		if err != nil {
			return nil, fmt.Errorf("invalid time value in cursor: %w", err)
		}

		return t, nil
	default:
		return value, nil
	}
}

// cursorTimeFormat is the time format used for cursor sort values.
const cursorTimeFormat = "2006-01-02T15:04:05.000Z"

// getExecutionSortValue extracts the sort field value from a workflow execution.
func getExecutionSortValue(e *model.WorkflowExecution, sortBy string) string {
	switch sortBy {
	case "completedAt":
		if e.CompletedAt() != nil {
			return e.CompletedAt().Format(cursorTimeFormat)
		}

		return ""
	default:
		return e.StartedAt().Format(cursorTimeFormat)
	}
}
