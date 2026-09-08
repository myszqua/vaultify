package vaultify

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
)

// Validate runs struct-tag validation (required/min/max/pattern) on target.
func Validate(target any) error {
	val := reflect.ValueOf(target)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	return validateStruct(val, "")
}

func validateStruct(val reflect.Value, prefix string) error {
	t := val.Type()

	var errs []*ValidationError

	for i := range t.NumField() {
		field := t.Field(i)
		fieldVal := val.Field(i)

		fieldPath := field.Name
		if prefix != "" {
			fieldPath = prefix + "." + field.Name
		}

		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				continue
			}

			fieldVal = fieldVal.Elem()
		}

		if fieldVal.Kind() == reflect.Struct {
			if err := validateStruct(fieldVal, fieldPath); err != nil {
				if validationErr, ok := err.(*ValidationError); ok { //nolint:errorlint
					errs = append(errs, validationErr)
				}
			}

			continue
		}

		if err := validateField(field, fieldVal, fieldPath); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func validateField(field reflect.StructField, val reflect.Value, path string) *ValidationError {
	required := field.Tag.Get("config")

	if required == "required" && isZeroValue(val) {
		return &ValidationError{
			Field:   path,
			Rule:    "required",
			Message: "field is required",
		}
	}

	if minStr := field.Tag.Get("min"); minStr != "" {
		if err := validateMin(val, minStr, path); err != nil {
			return err
		}
	}

	if maxStr := field.Tag.Get("max"); maxStr != "" {
		if err := validateMax(val, maxStr, path); err != nil {
			return err
		}
	}

	if pattern := field.Tag.Get("pattern"); pattern != "" {
		if err := validatePattern(val, pattern, path); err != nil {
			return err
		}
	}

	return nil
}

func validateMin(val reflect.Value, minStr, path string) *ValidationError { //nolint:cyclop
	switch val.Kind() {
	case reflect.Int, reflect.Int64:
		minVal, err := strconv.ParseInt(minStr, 10, 64) //nolint:mnd
		if err != nil {
			return &ValidationError{Field: path, Rule: "min", Message: err.Error()}
		}

		if val.Int() < minVal {
			return &ValidationError{
				Field:   path,
				Rule:    "min",
				Message: fmt.Sprintf("value must be >= %d", minVal),
			}
		}
	case reflect.Float64:
		minVal, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return &ValidationError{Field: path, Rule: "min", Message: err.Error()}
		}

		if val.Float() < minVal {
			return &ValidationError{
				Field:   path,
				Rule:    "min",
				Message: fmt.Sprintf("value must be >= %f", minVal),
			}
		}
	case reflect.String:
		minLen, err := strconv.Atoi(minStr)
		if err != nil {
			return &ValidationError{Field: path, Rule: "min", Message: err.Error()}
		}

		if val.Len() < minLen {
			return &ValidationError{
				Field:   path,
				Rule:    "min",
				Message: fmt.Sprintf("length must be >= %d", minLen),
			}
		}
	default:
		return nil
	}

	return nil
}

func validateMax(val reflect.Value, maxStr, path string) *ValidationError { //nolint:cyclop
	switch val.Kind() {
	case reflect.Int, reflect.Int64:
		maxVal, err := strconv.ParseInt(maxStr, 10, 64) //nolint:mnd
		if err != nil {
			return &ValidationError{Field: path, Rule: "max", Message: err.Error()}
		}

		if val.Int() > maxVal {
			return &ValidationError{
				Field:   path,
				Rule:    "max",
				Message: fmt.Sprintf("value must be <= %d", maxVal),
			}
		}
	case reflect.Float64:
		maxVal, err := strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return &ValidationError{Field: path, Rule: "max", Message: err.Error()}
		}

		if val.Float() > maxVal {
			return &ValidationError{
				Field:   path,
				Rule:    "max",
				Message: fmt.Sprintf("value must be <= %f", maxVal),
			}
		}
	case reflect.String:
		maxLen, err := strconv.Atoi(maxStr)
		if err != nil {
			return &ValidationError{Field: path, Rule: "max", Message: err.Error()}
		}

		if val.Len() > maxLen {
			return &ValidationError{
				Field:   path,
				Rule:    "max",
				Message: fmt.Sprintf("length must be <= %d", maxLen),
			}
		}
	default:
		return nil
	}

	return nil
}

func validatePattern(val reflect.Value, pattern, path string) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	matched, err := regexp.MatchString(pattern, val.String())
	if err != nil {
		return &ValidationError{
			Field:   path,
			Rule:    "pattern",
			Message: fmt.Sprintf("invalid pattern %q: %v", pattern, err),
		}
	}

	if !matched {
		return &ValidationError{
			Field:   path,
			Rule:    "pattern",
			Message: fmt.Sprintf("value does not match pattern %q", pattern),
		}
	}

	return nil
}

func isZeroValue(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return val.IsNil()
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64,
		reflect.Complex128, reflect.Chan, reflect.Func, reflect.Struct,
		reflect.UnsafePointer:
		return false
	default:
		return false
	}
}
