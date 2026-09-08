package vaultify

import (
	"fmt"
	"reflect"
)

type providerData struct {
	name     string
	priority int
	data     map[string]any
}

func deepMerge(base, override map[string]any, opts MergeOptions, logger Logger) map[string]any {
	result := make(map[string]any)

	for k, v := range base {
		result[k] = v
	}

	for key, overrideVal := range override {
		baseVal, exists := result[key]
		if !exists {
			result[key] = overrideVal

			continue
		}

		baseMap, baseIsMap := baseVal.(map[string]any)
		overrideMap, overrideIsMap := overrideVal.(map[string]any)

		if baseIsMap && overrideIsMap {
			result[key] = deepMerge(baseMap, overrideMap, opts, logger)

			continue
		}

		if baseIsMap != overrideIsMap {
			logger.Debug("type mismatch for key ", key, ": merging overwrites map with non-map value")

			result[key] = overrideVal

			continue
		}

		baseSlice, baseIsSlice := baseVal.([]any)
		overrideSlice, overrideIsSlice := overrideVal.([]any)

		if baseIsSlice && overrideIsSlice {
			result[key] = mergeSlices(baseSlice, overrideSlice, opts.SliceStrategy)

			continue
		}

		if !typesMatch(baseVal, overrideVal) {
			logger.Debug("type conflict for key ", key,
				": base=", reflect.TypeOf(baseVal),
				", override=", reflect.TypeOf(overrideVal),
				", using override value")
		}

		result[key] = overrideVal
	}

	return result
}

func mergeSlices(base, override []any, strategy MergeStrategy) any { //nolint:ireturn
	switch strategy {
	case MergeStrategyReplace:
		return override
	case MergeStrategyMerge:
		merged := make([]any, 0, len(base)+len(override))
		merged = append(merged, base...)

		for _, v := range override {
			if !containsAny(merged, v) {
				merged = append(merged, v)
			}
		}

		return merged
	default:
		return override
	}
}

func containsAny(slice []any, target any) bool {
	targetStr := fmt.Sprintf("%v", target)

	for _, v := range slice {
		if fmt.Sprintf("%v", v) == targetStr {
			return true
		}
	}

	return false
}

func typesMatch(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}

	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

func applyPriorities(providerDataList []providerData, opts MergeOptions, logger Logger) map[string]any {
	result := make(map[string]any)

	sorted := sortByPriority(providerDataList)

	for _, pd := range sorted {
		logger.Debug("merging provider ", pd.name, " (priority: ", pd.priority, ")")

		result = deepMerge(result, pd.data, opts, logger)
	}

	return result
}

func sortByPriority(data []providerData) []providerData {
	sorted := make([]providerData, len(data))
	copy(sorted, data)

	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].priority < sorted[j-1].priority; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	return sorted
}
