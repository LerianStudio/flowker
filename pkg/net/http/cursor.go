// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package http

import (
	"encoding/base64"
	"encoding/json"
)

// Cursor contains all information needed to resume pagination consistently.
// Per PROJECT_RULES.md Section 9 (Pagination).
type Cursor struct {
	ID         string `json:"id"` // ID of the last item returned
	SortValue  string `json:"sv"` // Value of the sort field for the last item
	SortBy     string `json:"sb"` // Field used for sorting
	SortOrder  string `json:"so"` // Sort direction: "ASC" or "DESC"
	PointsNext bool   `json:"pn"` // Direction indicator (true = next page)
}

// DecodeCursor decodes a base64 encoded cursor string.
func DecodeCursor(cursor string) (Cursor, error) {
	decodedCursor, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return Cursor{}, err
	}

	var cur Cursor

	if err := json.Unmarshal(decodedCursor, &cur); err != nil {
		return Cursor{}, err
	}

	return cur, nil
}

// EncodeCursor encodes a Cursor struct to a base64 string.
func EncodeCursor(cur Cursor) (string, error) {
	jsonData, err := json.Marshal(cur)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(jsonData), nil
}
