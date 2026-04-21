// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package pagination provides shared helpers for cursor-based pagination
// across MongoDB repositories. It centralizes the cursor time format and the
// cursor-value parsing rules so every repository applies the same conversion
// (string cursor -> BSON Date / string) before building filters.
package pagination

import (
	"fmt"
	"time"
)

// SortTimeFormat is the layout used when encoding and decoding time-based
// cursor sort values (createdAt, updatedAt). Repositories must use this
// constant for both encoding (getSortValue) and decoding (ParseSortValue)
// so the round-trip is lossless.
const SortTimeFormat = "2006-01-02T15:04:05.000Z"

// ParseSortValue converts a cursor sort value string into the value type
// expected by MongoDB when building `$or` filters for keyset pagination.
//
// For time-based fields (createdAt, updatedAt), the BSON field is a Date,
// so the string cursor must be parsed back to time.Time. Comparing a string
// cursor directly against a BSON Date yields incorrect ordering and breaks
// pagination after the first page.
//
// For any other field, the value is returned unchanged.
func ParseSortValue(value, sortBy string) (any, error) {
	switch sortBy {
	case "createdAt", "updatedAt":
		if value == "" {
			return time.Time{}, nil
		}

		t, err := time.Parse(SortTimeFormat, value)
		if err != nil {
			return nil, fmt.Errorf("invalid time value in cursor: %w", err)
		}

		return t, nil
	default:
		return value, nil
	}
}
