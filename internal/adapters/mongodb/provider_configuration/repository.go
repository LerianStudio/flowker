// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package providerconfiguration

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
	// CollectionName is the MongoDB collection name for provider configurations.
	CollectionName = "provider_configurations"
)

// MongoDBRepository implements command.ProviderConfigRepository using MongoDB.
// Supports both single-tenant (fallback) and multi-tenant (context-based) modes.
type MongoDBRepository struct {
	fallbackDB *mongo.Database // Fallback for single-tenant mode
}

// NewMongoDBRepository creates a new MongoDB repository for provider configurations.
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

// Verify MongoDBRepository implements command.ProviderConfigRepository
var _ command.ProviderConfigRepository = (*MongoDBRepository)(nil)

// Create persists a new provider configuration to MongoDB.
func (r *MongoDBRepository) Create(ctx context.Context, p *model.ProviderConfiguration) error {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	mongoModel := NewMongoDBModelFromEntity(p)

	_, err = collection.InsertOne(ctx, mongoModel)
	if err != nil {
		// Check for duplicate key error (unique name constraint)
		if mongo.IsDuplicateKeyError(err) {
			return constant.ErrProviderConfigDuplicateName
		}

		return fmt.Errorf("failed to create provider configuration: %w", err)
	}

	return nil
}

// FindByID retrieves a provider configuration by its ID.
func (r *MongoDBRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"providerConfigId": id.String()}

	var mongoModel MongoDBModel

	err = collection.FindOne(ctx, filter).Decode(&mongoModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, constant.ErrProviderConfigNotFound
		}

		return nil, fmt.Errorf("failed to find provider configuration by ID: %w", err)
	}

	return mongoModel.ToEntity()
}

// FindByName retrieves a provider configuration by its name.
func (r *MongoDBRepository) FindByName(ctx context.Context, name string) (*model.ProviderConfiguration, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"name": name}

	var mongoModel MongoDBModel

	err = collection.FindOne(ctx, filter).Decode(&mongoModel)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, constant.ErrProviderConfigNotFound
		}

		return nil, fmt.Errorf("failed to find provider configuration by name: %w", err)
	}

	return mongoModel.ToEntity()
}

// List retrieves provider configurations with pagination and optional filtering.
func (r *MongoDBRepository) List(ctx context.Context, filter command.ProviderConfigListFilter) (*command.ProviderConfigListResult, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Build filter
	mongoFilter := bson.M{}

	if filter.Status != nil {
		mongoFilter["status"] = string(*filter.Status)
	}

	if filter.ProviderID != nil {
		mongoFilter["providerId"] = *filter.ProviderID
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
				{sortField: sortValue, "providerConfigId": bson.M{"$gt": decodedCursor.ID}},
			}
		} else {
			mongoFilter["$or"] = []bson.M{
				{sortField: bson.M{"$lt": sortValue}},
				{sortField: sortValue, "providerConfigId": bson.M{"$lt": decodedCursor.ID}},
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
		{Key: "providerConfigId", Value: sortDirection},
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
		return nil, fmt.Errorf("failed to list provider configurations: %w", err)
	}
	defer mongoCursor.Close(ctx)

	var mongoModels []MongoDBModel

	if err := mongoCursor.All(ctx, &mongoModels); err != nil {
		return nil, fmt.Errorf("failed to decode provider configurations: %w", err)
	}

	// Determine if there are more results
	hasMore := len(mongoModels) > limit
	if hasMore {
		mongoModels = mongoModels[:limit]
	}

	// Convert to domain entities
	providerConfigs := make([]*model.ProviderConfiguration, len(mongoModels))
	for i, m := range mongoModels {
		entity, err := m.ToEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert provider configuration document: %w", err)
		}

		providerConfigs[i] = entity
	}

	// Build next cursor
	var nextCursor string

	if hasMore && len(providerConfigs) > 0 {
		lastConfig := providerConfigs[len(providerConfigs)-1]
		sortValue := getSortValue(lastConfig, filter.SortBy)

		cur := nethttp.Cursor{
			ID:         lastConfig.ID().String(),
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

	return &command.ProviderConfigListResult{
		Items:      providerConfigs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

// Update persists changes to an existing provider configuration.
// expectedStatus is included in the filter to prevent race conditions (atomic check-and-set).
func (r *MongoDBRepository) Update(ctx context.Context, p *model.ProviderConfiguration, expectedStatus model.ProviderConfigurationStatus) error {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"providerConfigId": p.ID().String()}
	if expectedStatus != "" {
		filter["status"] = string(expectedStatus)
	}

	mongoModel := NewMongoDBModelFromEntity(p)

	update := bson.M{
		"$set": bson.M{
			"name":        mongoModel.Name,
			"description": mongoModel.Description,
			"providerId":  mongoModel.ProviderID,
			"config":      mongoModel.Config,
			"status":      mongoModel.Status,
			"metadata":    mongoModel.Metadata,
			"updatedAt":   mongoModel.UpdatedAt,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		// Check for duplicate key error (unique name constraint)
		if mongo.IsDuplicateKeyError(err) {
			return constant.ErrProviderConfigDuplicateName
		}

		return fmt.Errorf("failed to update provider configuration: %w", err)
	}

	if result.MatchedCount == 0 {
		if expectedStatus != "" {
			return constant.ErrConflictStateChanged
		}

		return constant.ErrProviderConfigNotFound
	}

	return nil
}

// Delete removes a provider configuration by its ID.
// expectedStatus is included in the filter to prevent race conditions (atomic check-and-set).
func (r *MongoDBRepository) Delete(ctx context.Context, id uuid.UUID, expectedStatus model.ProviderConfigurationStatus) error {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"providerConfigId": id.String()}
	if expectedStatus != "" {
		filter["status"] = string(expectedStatus)
	}

	result, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete provider configuration: %w", err)
	}

	if result.DeletedCount == 0 {
		if expectedStatus != "" {
			return constant.ErrConflictStateChanged
		}

		return constant.ErrProviderConfigNotFound
	}

	return nil
}

// ExistsByName checks if a provider configuration with the given name exists.
func (r *MongoDBRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	collection, err := r.getCollection(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get collection: %w", err)
	}

	filter := bson.M{"name": name}

	count, err := collection.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check provider configuration existence: %w", err)
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

// getSortValue extracts the sort field value from a provider configuration.
func getSortValue(p *model.ProviderConfiguration, sortBy string) string {
	switch sortBy {
	case "createdAt":
		return p.CreatedAt().Format(pagination.SortTimeFormat)
	case "updatedAt":
		return p.UpdatedAt().Format(pagination.SortTimeFormat)
	case "name":
		return p.Name()
	default:
		return p.CreatedAt().Format(pagination.SortTimeFormat)
	}
}
