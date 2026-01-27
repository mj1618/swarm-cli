package label

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// keyRegex matches valid label keys: starts with letter, followed by alphanumeric, dots, hyphens, underscores, slashes
	// Max 63 characters. Allows Kubernetes-style keys like app.kubernetes.io/name
	keyRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._/-]{0,62}$`)
	// valueRegex matches valid label values: alphanumeric, dots, hyphens, underscores, slashes
	// Max 253 characters, can be empty
	valueRegex = regexp.MustCompile(`^[a-zA-Z0-9._/-]{0,253}$`)
)

// Parse parses a label string in the format "key=value" or "key".
// Returns key, value, error. If no value, returns empty string for value.
func Parse(s string) (string, string, error) {
	parts := strings.SplitN(s, "=", 2)
	key := parts[0]

	if key == "" {
		return "", "", fmt.Errorf("label key cannot be empty")
	}

	if !keyRegex.MatchString(key) {
		return "", "", fmt.Errorf("invalid label key %q: must start with a letter and contain only alphanumeric characters, dots, hyphens, underscores, or slashes (max 63 chars)", key)
	}

	if strings.HasPrefix(key, "swarm.") {
		return "", "", fmt.Errorf("label key cannot use reserved prefix 'swarm.'")
	}

	if len(parts) == 1 {
		return key, "", nil
	}

	value := parts[1]
	if !valueRegex.MatchString(value) {
		return "", "", fmt.Errorf("invalid label value %q: must contain only alphanumeric characters, dots, hyphens, underscores, or slashes (max 253 chars)", value)
	}

	return key, value, nil
}

// ParseMultiple parses multiple label strings into a map.
// Later values override earlier ones for the same key.
func ParseMultiple(labels []string) (map[string]string, error) {
	if len(labels) == 0 {
		return nil, nil
	}

	result := make(map[string]string)
	for _, l := range labels {
		key, value, err := Parse(l)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

// Match checks if an agent's labels match the filter labels.
// For filters with values, exact match is required.
// For filters without values (key only), label existence is checked.
func Match(agentLabels, filterLabels map[string]string) bool {
	if len(filterLabels) == 0 {
		return true
	}

	if agentLabels == nil {
		return false
	}

	for key, filterValue := range filterLabels {
		agentValue, exists := agentLabels[key]
		if !exists {
			return false
		}
		// If filter has a value, it must match exactly
		if filterValue != "" && agentValue != filterValue {
			return false
		}
	}
	return true
}

// Format formats labels for display as a comma-separated string.
// Returns "-" if no labels are present.
func Format(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}

	pairs := make([]string, 0, len(labels))
	for k, v := range labels {
		if v == "" {
			pairs = append(pairs, k)
		} else {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}

// Merge merges two label maps, with the second map taking precedence.
// Returns a new map (does not modify the inputs).
func Merge(base, override map[string]string) map[string]string {
	if base == nil && override == nil {
		return nil
	}

	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
