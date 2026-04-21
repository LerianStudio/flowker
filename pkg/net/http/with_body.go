// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en2 "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/LerianStudio/flowker/pkg"
	cn "github.com/LerianStudio/flowker/pkg/constant"
)

var UUIDPathParameters = []string{
	"id",
}

// DecodeHandlerFunc is a handler which works with withBody decorator.
// It receives a struct which was decoded by withBody decorator before.
// Ex: json -> withBody -> DecodeHandlerFunc.
type DecodeHandlerFunc func(p any, c *fiber.Ctx) error

// PayloadContextValue is a wrapper type used to keep Context.Locals safe.
type PayloadContextValue string

// ConstructorFunc representing a constructor of any type.
type ConstructorFunc func() any

// decoderHandler decodes payload coming from requests.
type decoderHandler struct {
	handler      DecodeHandlerFunc
	constructor  ConstructorFunc
	structSource any
}

func newOfType(s any) any {
	t := reflect.TypeOf(s)
	v := reflect.New(t.Elem())

	return v.Interface()
}

func WithBody(s any, h DecodeHandlerFunc) fiber.Handler {
	d := &decoderHandler{
		handler:      h,
		structSource: s,
	}

	return d.FiberHandlerFunc
}

// FiberHandlerFunc is a method on the decoderHandler struct. It decodes the incoming request's body to a Go struct,
// validates it, checks for any extraneous fields not defined in the struct, and finally calls the wrapped handler function.
func (d *decoderHandler) FiberHandlerFunc(c *fiber.Ctx) error {
	var s any

	if d.constructor != nil {
		s = d.constructor()
	} else {
		s = newOfType(d.structSource)
	}

	bodyBytes := c.Body() // Get the body bytes

	if err := json.Unmarshal(bodyBytes, s); err != nil {
		return fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	marshaled, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal decoded struct: %w", err)
	}

	var originalMap, marshaledMap map[string]any

	if err := json.Unmarshal(bodyBytes, &originalMap); err != nil {
		return fmt.Errorf("failed to unmarshal request body to map: %w", err)
	}

	if err := json.Unmarshal(marshaled, &marshaledMap); err != nil {
		return fmt.Errorf("failed to unmarshal marshaled struct to map: %w", err)
	}

	diffFields := findUnknownFields(originalMap, marshaledMap)

	if len(diffFields) > 0 {
		err := pkg.ValidateBadRequestFieldsError(pkg.FieldValidations{}, pkg.FieldValidations{}, "", diffFields)
		return libHTTP.Respond(c, fiber.StatusBadRequest, err)
	}

	if err := ValidateStruct(s); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, err)
	}

	c.Locals("fields", diffFields)

	parseMetadata(s, originalMap)

	return d.handler(s, c)
}

// findUnknownFields finds fields that are present in the original map but not in the marshaled map.
func findUnknownFields(original, marshaled map[string]any) map[string]any {
	diffFields := make(map[string]any)

	numKinds := libCommons.GetMapNumKinds()

	for key, value := range original {
		if numKinds[reflect.ValueOf(value).Kind()] && value == 0.0 {
			continue
		}

		marshaledValue, ok := marshaled[key]
		if !ok {
			diffFields[key] = value
			continue
		}

		if diff, hasDiff := compareFieldValues(value, marshaledValue); hasDiff {
			diffFields[key] = diff
		}
	}

	return diffFields
}

// compareFieldValues compares an original field value against its marshaled counterpart.
// Returns the diff value and true if a difference was found.
func compareFieldValues(original, marshaled any) (any, bool) {
	switch originalValue := original.(type) {
	case map[string]any:
		if marshaledMap, ok := marshaled.(map[string]any); ok {
			nestedDiff := findUnknownFields(originalValue, marshaledMap)
			if len(nestedDiff) > 0 {
				return nestedDiff, true
			}
		} else if !reflect.DeepEqual(originalValue, marshaled) {
			return original, true
		}

	case []any:
		if marshaledArray, ok := marshaled.([]any); ok {
			arrayDiff := compareSlices(originalValue, marshaledArray)
			if len(arrayDiff) > 0 {
				return arrayDiff, true
			}
		} else if !reflect.DeepEqual(originalValue, marshaled) {
			return original, true
		}

	default:
		if !reflect.DeepEqual(original, marshaled) {
			return original, true
		}
	}

	return nil, false
}

// compareSlices compares two slices and returns differences.
func compareSlices(original, marshaled []any) []any {
	var diff []any

	// Iterate through the original slice and check differences
	for i, item := range original {
		if i >= len(marshaled) {
			// If marshaled slice is shorter, the original item is missing
			diff = append(diff, item)
		} else {
			tmpMarshaled := marshaled[i]
			// Compare individual items at the same index
			if originalMap, ok := item.(map[string]any); ok {
				if marshaledMap, ok := tmpMarshaled.(map[string]any); ok {
					nestedDiff := findUnknownFields(originalMap, marshaledMap)
					if len(nestedDiff) > 0 {
						diff = append(diff, nestedDiff)
					}
				}
			} else if !reflect.DeepEqual(item, tmpMarshaled) {
				diff = append(diff, item)
			}
		}
	}

	// Check if marshaled slice is longer
	for i := len(original); i < len(marshaled); i++ {
		diff = append(diff, marshaled[i])
	}

	return diff
}

// ValidateStruct validates a struct against defined validation rules, using the validator package.
func ValidateStruct(s any) error {
	v, trans := newValidator()

	k := reflect.ValueOf(s).Kind()
	if k == reflect.Ptr {
		k = reflect.ValueOf(s).Elem().Kind()
	}

	if k != reflect.Struct {
		return nil
	}

	err := v.Struct(s)
	if err != nil {
		for _, fieldError := range err.(validator.ValidationErrors) {
			switch fieldError.Tag() {
			case "keymax":
				return pkg.ValidateBusinessError(cn.ErrMetadataKeyLengthExceeded, "", fieldError.Translate(trans), fieldError.Param())
			case "valuemax":
				return pkg.ValidateBusinessError(cn.ErrMetadataValueLengthExceeded, "", fieldError.Translate(trans), fieldError.Param())
			case "nonested":
				return pkg.ValidateBusinessError(cn.ErrInvalidMetadataNesting, "", fieldError.Translate(trans))
			}
		}

		errPtr := malformedRequestErr(err.(validator.ValidationErrors), trans)

		return &errPtr
	}

	return nil
}

func fields(errs validator.ValidationErrors, trans ut.Translator) pkg.FieldValidations {
	l := len(errs)
	if l > 0 {
		fields := make(pkg.FieldValidations, l)
		for _, e := range errs {
			fields[e.Field()] = e.Translate(trans)
		}

		return fields
	}

	return nil
}

func fieldsRequired(myMap pkg.FieldValidations) pkg.FieldValidations {
	result := make(pkg.FieldValidations)

	for key, value := range myMap {
		if strings.Contains(value, "required") {
			result[key] = value
		}
	}

	return result
}

func malformedRequestErr(err validator.ValidationErrors, trans ut.Translator) pkg.ValidationKnownFieldsError {
	invalidFieldsMap := fields(err, trans)

	requiredFields := fieldsRequired(invalidFieldsMap)

	var vErr pkg.ValidationKnownFieldsError

	_ = errors.As(pkg.ValidateBadRequestFieldsError(requiredFields, invalidFieldsMap, "", make(map[string]any)), &vErr)

	return vErr
}

// newValidator creates a new validator instance with translations.
// This function uses panic intentionally for RegisterDefaultTranslations failure
// as this is critical bootstrap code - if validation cannot be configured,
// the application should not start.
//
//nolint:ireturn
func newValidator() (*validator.Validate, ut.Translator) {
	locale := en.New()
	uni := ut.New(locale, locale)

	trans, found := uni.GetTranslator("en")
	if !found {
		// This should never happen as we just registered the "en" locale
		panic("failed to get English translator")
	}

	v := validator.New()

	if err := en2.RegisterDefaultTranslations(v, trans); err != nil {
		panic(err)
	}

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}

		return name
	})

	// Register custom validations - these only fail if the tag name conflicts
	// with an existing registration, which is a programming error.
	mustRegisterValidation(v, "keymax", validateMetadataKeyMaxLength)
	mustRegisterValidation(v, "nonested", validateMetadataNestedValues)
	mustRegisterValidation(v, "valuemax", validateMetadataValueMaxLength)

	// Register custom translations - these only fail if the translation key
	// conflicts with an existing registration, which is a programming error.
	mustRegisterTranslation(v, trans, "required", "{0} is a required field")
	mustRegisterTranslation(v, trans, "gte", "{0} must be {1} or greater")
	mustRegisterTranslation(v, trans, "eq", "{0} is not equal to {1}")
	mustRegisterTranslation(v, trans, "keymax", "{0}")
	mustRegisterTranslation(v, trans, "valuemax", "{0}")
	mustRegisterTranslation(v, trans, "nonested", "{0}")

	return v, trans
}

// validateMetadataNestedValues checks if there are nested metadata structures
func validateMetadataNestedValues(fl validator.FieldLevel) bool {
	return fl.Field().Kind() != reflect.Map
}

// validateMetadataKeyMaxLength checks if metadata key (always a string) length is allowed
func validateMetadataKeyMaxLength(fl validator.FieldLevel) bool {
	limitParam := fl.Param()

	limit := 100 // default limit if no param configured

	if limitParam != "" {
		if parsedParam, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedParam
		}
	}

	return len(fl.Field().String()) <= limit
}

// validateMetadataValueMaxLength checks metadata value max length
func validateMetadataValueMaxLength(fl validator.FieldLevel) bool {
	limitParam := fl.Param()

	limit := 2000 // default limit if no param configured

	if limitParam != "" {
		if parsedParam, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedParam
		}
	}

	var value string

	switch fl.Field().Kind() {
	case reflect.Int:
		value = strconv.Itoa(int(fl.Field().Int()))
	case reflect.Float64:
		value = strconv.FormatFloat(fl.Field().Float(), 'f', -1, 64)
	case reflect.String:
		value = fl.Field().String()
	case reflect.Bool:
		value = strconv.FormatBool(fl.Field().Bool())
	default:
		return false
	}

	return len(value) <= limit
}

// fieldNameRegex is pre-compiled for performance and to avoid runtime compilation errors.
var fieldNameRegex = regexp.MustCompile(`\.(.+)$`)

// formatErrorFieldName formats metadata field error names for error messages
func formatErrorFieldName(text string) string {
	matches := fieldNameRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	return text
}

// mustRegisterValidation registers a validation and panics if it fails.
// This is acceptable because registration failures are programming errors
// that should be caught during development, not runtime.
func mustRegisterValidation(v *validator.Validate, tag string, fn validator.Func) {
	if err := v.RegisterValidation(tag, fn); err != nil {
		panic("failed to register validation " + tag + ": " + err.Error())
	}
}

// mustRegisterTranslation registers a translation with proper error handling.
// The translation function wraps the common pattern of registering a simple message.
func mustRegisterTranslation(v *validator.Validate, trans ut.Translator, tag, message string) {
	registerFn := func(translator ut.Translator) error {
		return translator.Add(tag, message, true)
	}

	translateFn := func(translator ut.Translator, fe validator.FieldError) string {
		t, _ := translator.T(tag, formatErrorFieldName(fe.Namespace()), fe.Param())
		return t
	}

	if err := v.RegisterTranslation(tag, trans, registerFn, translateFn); err != nil {
		panic("failed to register translation " + tag + ": " + err.Error())
	}
}

// parseMetadata For compliance with RFC7396 JSON Merge Patch
func parseMetadata(s any, originalMap map[string]any) {
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return
	}

	val = val.Elem()

	metadataField := val.FieldByName("Metadata")
	if !metadataField.IsValid() || !metadataField.CanSet() {
		return
	}

	if _, exists := originalMap["metadata"]; !exists {
		metadataField.Set(reflect.ValueOf(make(map[string]any)))
	}
}

// ParseUUIDPathParameters globally, considering all path parameters are UUIDs
func ParseUUIDPathParameters(c *fiber.Ctx) error {
	params := c.AllParams()

	var invalidUUIDs []string

	validPathParamsMap := make(map[string]any)

	for param, value := range params {
		if !libCommons.Contains[string](UUIDPathParameters, param) {
			validPathParamsMap[param] = value
			continue
		}

		parsedUUID, err := uuid.Parse(value)
		if err != nil {
			invalidUUIDs = append(invalidUUIDs, param)
			continue
		}

		validPathParamsMap[param] = parsedUUID
	}

	for param, value := range validPathParamsMap {
		c.Locals(param, value)
	}

	if len(invalidUUIDs) > 0 {
		err := pkg.ValidateBusinessError(cn.ErrInvalidPathParameter, "", strings.Join(invalidUUIDs, ", "))
		return WithError(c, err)
	}

	return c.Next()
}
