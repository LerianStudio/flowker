// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/LerianStudio/flowker/internal/services/command"
	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	nethttp "github.com/LerianStudio/flowker/pkg/net/http"
	"github.com/LerianStudio/flowker/pkg/pagination"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// CollectionName is the MongoDB collection name for workflows.
	CollectionName = "workflows"
)

// MongoDBRepository implements command.WorkflowRepository using MongoDB.
// Supports both single-tenant (fallback) and multi-tenant (context-based) modes.
type MongoDBRepository struct {
	fallbackDB *mongo.Database // Fallback for single-tenant mode
}

// NewMongoDBRepository creates a new MongoDB repository for workflows.
// The provided database is used as fallback in single-tenant mode.
// In multi-tenant mode, the database is resolved from context.
func NewMongoDBRepository(db *mongo.Database) *MongoDBRepository {
	return &MongoDBRepository{
		fallbackDB: db,
	}
}

// getCollection returns the tenant-specific collection or fallback.
// In multi-tenant mode, it extracts the database from context.
// In single-tenant mode, it uses the fallback database.
func (r *MongoDBRepository) getCollection(ctx context.Context) (*mongo.Collection, error) {
	// Try to get tenant-specific database from context (multi-tenant mode)
	db := tmcore.GetMBContext(ctx)
	if db != nil {
		return db.Collection(CollectionName), nil
	}

	// Single-tenant mode: use fallback
	if r.fallbackDB == nil {
		return nil, errors.New("mongodb connection not available")
	}

	return r.fallbackDB.Collection(CollectionName), nil
}

// Verify MongoDBRepository implements command.WorkflowRepository
var _ command.WorkflowRepository = (*MongoDBRepository)(nil)

// Create persists a new workflow to MongoDB.
func (r *MongoDBRepository) Create(ctx context.Context, w *model.Workflow) error {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	mongoModel := NewMongoDBModelFromEntity(w)

	_, err = collection.InsertOne(ctx, mongoModel)
	if err != nil {
		// Check for duplicate key error (unique name constraint)
		if mongo.IsDuplicateKeyError(err) {
			return constant.ErrWorkflowDuplicateName
		}

		return fmt.Errorf("failed to create workflow: %w", err)
	}

	return nil
}

// FindByID retrieves a workflow by its ID.
func (r *MongoDBRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"workflowId": id.String()}

	var mongoModel MongoDBModel

	err = collection.FindOne(ctx, filter).Decode(&mongoModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, constant.ErrWorkflowNotFound
		}

		return nil, fmt.Errorf("failed to find workflow by ID: %w", err)
	}

	return mongoModel.ToEntity(), nil
}

// FindByName retrieves a workflow by its name.
func (r *MongoDBRepository) FindByName(ctx context.Context, name string) (*model.Workflow, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"name": name}

	var mongoModel MongoDBModel

	err = collection.FindOne(ctx, filter).Decode(&mongoModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, constant.ErrWorkflowNotFound
		}

		return nil, fmt.Errorf("failed to find workflow by name: %w", err)
	}

	return mongoModel.ToEntity(), nil
}

// List retrieves workflows with pagination and optional filtering.
func (r *MongoDBRepository) List(ctx context.Context, filter command.WorkflowListFilter) (*command.WorkflowListResult, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Build filter
	mongoFilter := bson.M{}

	if filter.Status != nil {
		mongoFilter["status"] = string(*filter.Status)
	}

	// Apply cursor if provided
	if filter.Cursor != "" {
		decodedCursor, err := nethttp.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, pkg.ValidationError{
				Code:    "INVALID_CURSOR",
				Message: "invalid cursor format",
				Err:     err,
			}
		}

		// Build cursor condition based on sort order
		sortField := mapSortField(filter.SortBy)

		sortValue, err := pagination.ParseSortValue(decodedCursor.SortValue, filter.SortBy)
		if err != nil {
			return nil, pkg.ValidationError{
				Code:    "INVALID_CURSOR",
				Message: "invalid cursor sort value",
				Err:     err,
			}
		}

		if filter.SortOrder == "ASC" {
			mongoFilter["$or"] = []bson.M{
				{sortField: bson.M{"$gt": sortValue}},
				{sortField: sortValue, "workflowId": bson.M{"$gt": decodedCursor.ID}},
			}
		} else {
			mongoFilter["$or"] = []bson.M{
				{sortField: bson.M{"$lt": sortValue}},
				{sortField: sortValue, "workflowId": bson.M{"$lt": decodedCursor.ID}},
			}
		}
	}

	// Build sort options
	sortDirection := -1 // DESC
	if filter.SortOrder == "ASC" {
		sortDirection = 1
	}

	sortField := mapSortField(filter.SortBy)
	sortOpts := bson.D{
		{Key: sortField, Value: sortDirection},
		{Key: "workflowId", Value: sortDirection},
	}

	// Fetch limit + 1 to determine if there are more results
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}

	findOptions := options.Find().
		SetSort(sortOpts).
		SetLimit(int64(limit + 1))

	mongoCursor, err := collection.Find(ctx, mongoFilter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer mongoCursor.Close(ctx)

	var mongoModels []MongoDBModel

	if err := mongoCursor.All(ctx, &mongoModels); err != nil {
		return nil, fmt.Errorf("failed to decode workflows: %w", err)
	}

	// Determine if there are more results
	hasMore := len(mongoModels) > limit
	if hasMore {
		mongoModels = mongoModels[:limit]
	}

	// Convert to domain entities
	workflows := make([]*model.Workflow, len(mongoModels))
	for i, m := range mongoModels {
		workflows[i] = m.ToEntity()
	}

	// Build next cursor
	var nextCursor string

	if hasMore && len(workflows) > 0 {
		lastWorkflow := workflows[len(workflows)-1]
		sortValue := getSortValue(lastWorkflow, filter.SortBy)

		cur := nethttp.Cursor{
			ID:         lastWorkflow.ID().String(),
			SortValue:  sortValue,
			SortBy:     filter.SortBy,
			SortOrder:  filter.SortOrder,
			PointsNext: true,
		}

		encoded, err := nethttp.EncodeCursor(cur)
		if err != nil {
			return nil, fmt.Errorf("failed to encode cursor: %w", err)
		}

		nextCursor = encoded
	}

	return &command.WorkflowListResult{
		Items:      workflows,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// Update persists changes to an existing workflow.
// expectedStatus is included in the filter to prevent race conditions (atomic check-and-set).
func (r *MongoDBRepository) Update(ctx context.Context, w *model.Workflow, expectedStatus model.WorkflowStatus) error {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"workflowId": w.ID().String()}
	if expectedStatus != "" {
		filter["status"] = string(expectedStatus)
	}

	mongoModel := NewMongoDBModelFromEntity(w)

	update := bson.M{
		"$set": bson.M{
			"name":        mongoModel.Name,
			"description": mongoModel.Description,
			"status":      mongoModel.Status,
			"nodes":       mongoModel.Nodes,
			"edges":       mongoModel.Edges,
			"metadata":    mongoModel.Metadata,
			"updatedAt":   mongoModel.UpdatedAt,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		// Check for duplicate key error (unique name constraint)
		if mongo.IsDuplicateKeyError(err) {
			return constant.ErrWorkflowDuplicateName
		}

		return fmt.Errorf("failed to update workflow: %w", err)
	}

	if result.MatchedCount == 0 {
		if expectedStatus != "" {
			return constant.ErrConflictStateChanged
		}

		return constant.ErrWorkflowNotFound
	}

	return nil
}

// Delete removes a workflow by its ID.
// expectedStatus is included in the filter to prevent race conditions (atomic check-and-set).
func (r *MongoDBRepository) Delete(ctx context.Context, id uuid.UUID, expectedStatus model.WorkflowStatus) error {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"workflowId": id.String()}
	if expectedStatus != "" {
		filter["status"] = string(expectedStatus)
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	if result.DeletedCount == 0 {
		if expectedStatus != "" {
			return constant.ErrConflictStateChanged
		}

		return constant.ErrWorkflowNotFound
	}

	return nil
}

// ExistsByName checks if a workflow with the given name exists.
func (r *MongoDBRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"name": name}

	count, err := collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check workflow existence: %w", err)
	}

	return count > 0, nil
}

// mapSortField maps API field names to MongoDB field names.
func mapSortField(field string) string {
	switch field {
	case "createdAt":
		return "createdAt"
	case "updatedAt":
		return "updatedAt"
	case "name":
		return "name"
	default:
		return "createdAt"
	}
}

// getSortValue extracts the sort field value from a workflow.
func getSortValue(w *model.Workflow, sortBy string) string {
	switch sortBy {
	case "createdAt":
		return w.CreatedAt().Format(pagination.SortTimeFormat)
	case "updatedAt":
		return w.UpdatedAt().Format(pagination.SortTimeFormat)
	case "name":
		return w.Name()
	default:
		return w.CreatedAt().Format(pagination.SortTimeFormat)
	}
}
