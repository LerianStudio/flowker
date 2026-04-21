// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package model contains domain entities and DTOs for Flowker.
package model

import (
	"time"

	"github.com/google/uuid"
)

// AuditActorOutput is the output DTO for an audit actor.
type AuditActorOutput struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	IPAddress string `json:"ipAddress"`
}

// AuditEntryOutput is the output DTO for a single audit entry.
// @Description Immutable audit event record with hash chain integrity.
type AuditEntryOutput struct {
	EventID      uuid.UUID        `json:"eventId" swaggertype:"string" format:"uuid"`
	EventType    string           `json:"eventType"`
	Action       string           `json:"action"`
	Result       string           `json:"result"`
	ResourceID   string           `json:"resourceId"`
	ResourceType string           `json:"resourceType"`
	Actor        AuditActorOutput `json:"actor"`
	Context      map[string]any   `json:"context,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
	Timestamp    time.Time        `json:"timestamp"`
	Hash         string           `json:"hash"`
	PreviousHash string           `json:"previousHash,omitempty"`
}

// AuditEntryListOutput is the output DTO for listing audit entries.
type AuditEntryListOutput struct {
	Items      []AuditEntryOutput `json:"items"`
	NextCursor string             `json:"nextCursor"`
	HasMore    bool               `json:"hasMore"`
}

// HashChainVerificationOutput is the output DTO for hash chain verification.
// @Description Result of hash chain integrity verification.
type HashChainVerificationOutput struct {
	IsValid        bool   `json:"isValid"`
	FirstInvalidID *int64 `json:"firstInvalidId,omitempty"`
	TotalChecked   int64  `json:"totalChecked"`
	Message        string `json:"message"`
}

// AuditEntryOutputFromDomain converts an AuditEntry domain entity to AuditEntryOutput.
func AuditEntryOutputFromDomain(entry *AuditEntry) AuditEntryOutput {
	if entry == nil {
		return AuditEntryOutput{}
	}

	return AuditEntryOutput{
		EventID:      entry.EventID(),
		EventType:    string(entry.EventType()),
		Action:       string(entry.Action()),
		Result:       string(entry.Result()),
		ResourceID:   entry.ResourceID(),
		ResourceType: string(entry.ResourceType()),
		Actor: AuditActorOutput{
			Type:      string(entry.Actor().Type()),
			ID:        entry.Actor().ID(),
			IPAddress: entry.Actor().IPAddress(),
		},
		Context:      entry.Context(),
		Metadata:     entry.Metadata(),
		Timestamp:    entry.Timestamp(),
		Hash:         entry.Hash(),
		PreviousHash: entry.PreviousHash(),
	}
}

// AuditEntryListOutputFromDomain converts a list of AuditEntry domain entities to list output.
func AuditEntryListOutputFromDomain(entries []*AuditEntry, nextCursor string, hasMore bool) AuditEntryListOutput {
	items := make([]AuditEntryOutput, 0, len(entries))
	for _, entry := range entries {
		if entry != nil {
			items = append(items, AuditEntryOutputFromDomain(entry))
		}
	}

	return AuditEntryListOutput{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
