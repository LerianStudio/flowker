// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package dashboard contains the MongoDB repository for dashboard aggregation queries.
package dashboard

import (
	"context"
	"fmt"

	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Compile-time interface check.
var _ query.DashboardRepository = (*MongoDBRepository)(nil)

type statusCount struct {
	Status string `bson:"_id"`
	Count  int64  `bson:"count"`
}

// MongoDBRepository implements DashboardRepository using MongoDB aggregation pipelines.
type MongoDBRepository struct {
	workflowCol  *mongo.Collection
	executionCol *mongo.Collection
}

// NewMongoDBRepository creates a new MongoDB repository for dashboard queries.
func NewMongoDBRepository(db *mongo.Database) (*MongoDBRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database cannot be nil")
	}

	return &MongoDBRepository{
		workflowCol:  db.Collection("workflows"),
		executionCol: db.Collection("workflow_executions"),
	}, nil
}

// WorkflowSummary aggregates workflow counts grouped by status.
func (r *MongoDBRepository) WorkflowSummary(ctx context.Context) (*model.WorkflowSummaryOutput, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err := r.workflowCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate workflow summary: %w", err)
	}
	defer cursor.Close(ctx)

	var counts []statusCount
	if err := cursor.All(ctx, &counts); err != nil {
		return nil, fmt.Errorf("failed to decode workflow summary: %w", err)
	}

	output := &model.WorkflowSummaryOutput{
		ByStatus: make([]model.StatusCountOutput, 0, len(counts)),
	}

	for _, c := range counts {
		output.Total += c.Count

		if c.Status == string(model.WorkflowStatusActive) {
			output.Active = c.Count
		}

		output.ByStatus = append(output.ByStatus, model.StatusCountOutput{
			Status: c.Status,
			Count:  c.Count,
		})
	}

	return output, nil
}

// ExecutionSummary aggregates execution counts grouped by status with optional filters.
func (r *MongoDBRepository) ExecutionSummary(ctx context.Context, filter query.ExecutionSummaryFilter) (*model.ExecutionSummaryOutput, error) {
	matchStage := bson.D{}

	if filter.StartTime != nil || filter.EndTime != nil {
		startedAt := bson.D{}
		if filter.StartTime != nil {
			startedAt = append(startedAt, bson.E{Key: "$gte", Value: *filter.StartTime})
		}
		if filter.EndTime != nil {
			startedAt = append(startedAt, bson.E{Key: "$lte", Value: *filter.EndTime})
		}
		matchStage = append(matchStage, bson.E{Key: "startedAt", Value: startedAt})
	}

	if filter.Status != nil {
		matchStage = append(matchStage, bson.E{Key: "status", Value: *filter.Status})
	}

	pipeline := mongo.Pipeline{}

	if len(matchStage) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchStage}})
	}

	pipeline = append(pipeline, bson.D{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: "$status"},
		{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
	}}})

	cursor, err := r.executionCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate execution summary: %w", err)
	}
	defer cursor.Close(ctx)

	var counts []statusCount
	if err := cursor.All(ctx, &counts); err != nil {
		return nil, fmt.Errorf("failed to decode execution summary: %w", err)
	}

	output := &model.ExecutionSummaryOutput{}

	for _, c := range counts {
		output.Total += c.Count

		switch model.ExecutionStatus(c.Status) {
		case model.ExecutionStatusCompleted:
			output.Completed = c.Count
		case model.ExecutionStatusFailed:
			output.Failed = c.Count
		case model.ExecutionStatusPending:
			output.Pending = c.Count
		case model.ExecutionStatusRunning:
			output.Running = c.Count
		}
	}

	return output, nil
}
