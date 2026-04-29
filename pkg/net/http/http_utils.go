// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package http

import (
	"strconv"
	"strings"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
)

// QueryHeader entity from query parameter from get apis
// Note: Cursor-based pagination only (no offset/page-based pagination per PROJECT_RULES.md)
type QueryHeader struct {
	Metadata    *bson.M
	Limit       int
	Cursor      string
	SortOrder   string
	StartDate   time.Time
	EndDate     time.Time
	UseMetadata bool
}

// Pagination entity for cursor-based pagination
type Pagination struct {
	Limit     int
	Cursor    string
	SortOrder string
	StartDate time.Time
	EndDate   time.Time
}

// ToCursorPagination converts QueryHeader to cursor-based Pagination
func (qh *QueryHeader) ToCursorPagination() Pagination {
	return Pagination{
		Limit:     qh.Limit,
		Cursor:    qh.Cursor,
		SortOrder: qh.SortOrder,
		StartDate: qh.StartDate,
		EndDate:   qh.EndDate,
	}
}

// ValidateParameters validate and return struct of default parameters
func ValidateParameters(params map[string]string) (*QueryHeader, error) {
	var (
		metadata    *bson.M
		startDate   time.Time
		endDate     time.Time
		cursor      string
		limit       = 10
		sortOrder   = "desc"
		useMetadata = false
	)

	for key, value := range params {
		switch {
		case strings.Contains(key, "metadata."):
			metadata = &bson.M{key: value}
			useMetadata = true
		case strings.Contains(key, "limit"):
			limitVal, err := strconv.Atoi(value)
			if err != nil {
				return nil, pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, "", "limit")
			}

			limit = limitVal
		case strings.Contains(key, "cursor"):
			cursor = value
		case strings.Contains(key, "sort_order"):
			sortOrder = strings.ToLower(value)
		case strings.Contains(key, "start_date"):
			parsedDate, err := time.Parse("2006-01-02", value)
			if err != nil {
				return nil, pkg.ValidateBusinessError(constant.ErrInvalidDateFormat, "", "start_date")
			}

			startDate = parsedDate
		case strings.Contains(key, "end_date"):
			parsedDate, err := time.Parse("2006-01-02", value)
			if err != nil {
				return nil, pkg.ValidateBusinessError(constant.ErrInvalidDateFormat, "", "end_date")
			}

			endDate = parsedDate
		}
	}

	err := validateDates(&startDate, &endDate)
	if err != nil {
		return nil, err
	}

	err = validatePagination(cursor, sortOrder, limit)
	if err != nil {
		return nil, err
	}

	query := &QueryHeader{
		Metadata:    metadata,
		Limit:       limit,
		Cursor:      cursor,
		SortOrder:   sortOrder,
		StartDate:   startDate,
		EndDate:     endDate,
		UseMetadata: useMetadata,
	}

	return query, nil
}

func validateDates(startDate, endDate *time.Time) error {
	maxDateRangeMonths := libCommons.SafeInt64ToInt(pkg.GetenvIntOrDefault("MAX_PAGINATION_MONTH_DATE_RANGE", 1))

	defaultStartDate := time.Now().AddDate(0, -maxDateRangeMonths, 0)
	defaultEndDate := time.Now()

	if !startDate.IsZero() && !endDate.IsZero() {
		if !pkg.IsValidDate(pkg.NormalizeDate(*startDate, nil)) || !pkg.IsValidDate(pkg.NormalizeDate(*endDate, nil)) {
			return pkg.ValidateBusinessError(constant.ErrInvalidDateFormat, "")
		}

		if !pkg.IsInitialDateBeforeFinalDate(*startDate, *endDate) {
			return pkg.ValidateBusinessError(constant.ErrInvalidFinalDate, "")
		}

		if !pkg.IsDateRangeWithinMonthLimit(*startDate, *endDate, maxDateRangeMonths) {
			return pkg.ValidateBusinessError(constant.ErrDateRangeExceedsLimit, "", maxDateRangeMonths)
		}
	}

	if startDate.IsZero() && endDate.IsZero() {
		*startDate = defaultStartDate
		*endDate = defaultEndDate
	}

	if (!startDate.IsZero() && endDate.IsZero()) ||
		(startDate.IsZero() && !endDate.IsZero()) {
		return pkg.ValidateBusinessError(constant.ErrInvalidDateRange, "")
	}

	return nil
}

func validatePagination(cursor, sortOrder string, limit int) error {
	maxPaginationLimit := libCommons.SafeInt64ToInt(pkg.GetenvIntOrDefault("MAX_PAGINATION_LIMIT", 100))

	if limit > maxPaginationLimit {
		return pkg.ValidateBusinessError(constant.ErrPaginationLimitExceeded, "", maxPaginationLimit)
	}

	if (sortOrder != string(constant.Asc)) && (sortOrder != string(constant.Desc)) {
		return pkg.ValidateBusinessError(constant.ErrInvalidSortOrder, "")
	}

	if !libCommons.IsNilOrEmpty(&cursor) {
		_, err := DecodeCursor(cursor)
		if err != nil {
			return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, "", "cursor")
		}
	}

	return nil
}
