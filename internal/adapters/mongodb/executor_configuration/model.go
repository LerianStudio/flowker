// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package executorconfiguration contains the MongoDB adapter for executor configuration persistence.
package executorconfiguration

import (
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MongoDBModel is the MongoDB document model for executor configurations.
// Implements ToEntity/FromEntity pattern per PROJECT_RULES.md.
type MongoDBModel struct {
	ObjectID       primitive.ObjectID  `bson:"_id,omitempty"`
	ExecutorID     string              `bson:"executorId"`
	Name           string              `bson:"name"`
	Description    *string             `bson:"description,omitempty"`
	BaseURL        string              `bson:"baseUrl"`
	Endpoints      []EndpointModel     `bson:"endpoints"`
	Authentication AuthenticationModel `bson:"authentication"`
	Status         string              `bson:"status"`
	Metadata       map[string]any      `bson:"metadata,omitempty"`
	CreatedAt      time.Time           `bson:"createdAt"`
	UpdatedAt      time.Time           `bson:"updatedAt"`
	LastTestedAt   *time.Time          `bson:"lastTestedAt,omitempty"`
}

// EndpointModel is the MongoDB model for executor endpoints.
type EndpointModel struct {
	Name    string `bson:"name"`
	Path    string `bson:"path"`
	Method  string `bson:"method"`
	Timeout int    `bson:"timeout"`
}

// AuthenticationModel is the MongoDB model for executor authentication.
type AuthenticationModel struct {
	Type   string         `bson:"type"`
	Config map[string]any `bson:"config,omitempty"`
}

// ToEntity converts MongoDBModel to domain entity.
func (m *MongoDBModel) ToEntity() *model.ExecutorConfiguration {
	executorID, _ := uuid.Parse(m.ExecutorID)

	endpoints := make([]model.ExecutorEndpoint, len(m.Endpoints))
	for i, epModel := range m.Endpoints {
		endpoints[i] = epModel.ToEntity()
	}

	authentication := m.Authentication.ToEntity()

	return model.NewExecutorConfigurationFromDB(
		executorID,
		m.Name,
		m.Description,
		m.BaseURL,
		endpoints,
		authentication,
		model.ExecutorConfigurationStatus(m.Status),
		m.Metadata,
		m.CreatedAt,
		m.UpdatedAt,
		m.LastTestedAt,
	)
}

// FromEntity populates MongoDBModel from domain entity.
func (m *MongoDBModel) FromEntity(p *model.ExecutorConfiguration) {
	m.ExecutorID = p.ID().String()
	m.Name = p.Name()
	m.Description = p.Description()
	m.BaseURL = p.BaseURL()
	m.Status = string(p.Status())
	m.Metadata = p.Metadata()
	m.CreatedAt = p.CreatedAt()
	m.UpdatedAt = p.UpdatedAt()
	m.LastTestedAt = p.LastTestedAt()

	m.Endpoints = make([]EndpointModel, len(p.Endpoints()))
	for i, ep := range p.Endpoints() {
		m.Endpoints[i] = EndpointModelFromEntity(ep)
	}

	m.Authentication = AuthenticationModelFromEntity(p.Authentication())
}

// ToEntity converts EndpointModel to domain entity.
func (e *EndpointModel) ToEntity() model.ExecutorEndpoint {
	return model.NewExecutorEndpointFromDB(
		e.Name,
		e.Path,
		e.Method,
		e.Timeout,
	)
}

// EndpointModelFromEntity creates an EndpointModel from domain entity.
func EndpointModelFromEntity(ep model.ExecutorEndpoint) EndpointModel {
	return EndpointModel{
		Name:    ep.Name(),
		Path:    ep.Path(),
		Method:  ep.Method(),
		Timeout: ep.Timeout(),
	}
}

// ToEntity converts AuthenticationModel to domain entity.
func (a *AuthenticationModel) ToEntity() model.ExecutorAuthentication {
	return model.NewExecutorAuthenticationFromDB(a.Type, a.Config)
}

// AuthenticationModelFromEntity creates an AuthenticationModel from domain entity.
func AuthenticationModelFromEntity(auth model.ExecutorAuthentication) AuthenticationModel {
	return AuthenticationModel{
		Type:   auth.Type(),
		Config: auth.Config(),
	}
}

// NewMongoDBModelFromEntity creates a new MongoDBModel from domain entity.
func NewMongoDBModelFromEntity(p *model.ExecutorConfiguration) *MongoDBModel {
	m := &MongoDBModel{}
	m.FromEntity(p)

	return m
}
