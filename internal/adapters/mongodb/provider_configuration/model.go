// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package providerconfiguration contains the MongoDB adapter for provider configuration persistence.
package providerconfiguration

import (
	"fmt"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MongoDBModel is the MongoDB document model for provider configurations.
// Implements ToEntity/FromEntity pattern per PROJECT_RULES.md.
type MongoDBModel struct {
	ObjectID         primitive.ObjectID `bson:"_id,omitempty"`
	ProviderConfigID string             `bson:"providerConfigId"`
	Name             string             `bson:"name"`
	Description      *string            `bson:"description,omitempty"`
	ProviderID       string             `bson:"providerId"`
	Config           map[string]any     `bson:"config"`
	Status           string             `bson:"status"`
	Metadata         map[string]any     `bson:"metadata,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
}

// ToEntity converts MongoDBModel to domain entity.
func (m *MongoDBModel) ToEntity() (*model.ProviderConfiguration, error) {
	providerConfigID, err := uuid.Parse(m.ProviderConfigID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider config ID %q: %w", m.ProviderConfigID, err)
	}

	return model.ReconstructProviderConfiguration(
		providerConfigID,
		m.Name,
		m.Description,
		m.ProviderID,
		m.Config,
		model.ProviderConfigurationStatus(m.Status),
		m.Metadata,
		m.CreatedAt,
		m.UpdatedAt,
	), nil
}

// FromEntity populates MongoDBModel from domain entity.
func (m *MongoDBModel) FromEntity(p *model.ProviderConfiguration) {
	m.ProviderConfigID = p.ID().String()
	m.Name = p.Name()
	m.Description = p.Description()
	m.ProviderID = p.ProviderID()
	m.Config = p.Config()
	m.Status = string(p.Status())
	m.Metadata = p.Metadata()
	m.CreatedAt = p.CreatedAt()
	m.UpdatedAt = p.UpdatedAt()
}

// NewMongoDBModelFromEntity creates a new MongoDBModel from domain entity.
func NewMongoDBModelFromEntity(p *model.ProviderConfiguration) *MongoDBModel {
	m := &MongoDBModel{}
	m.FromEntity(p)

	return m
}
