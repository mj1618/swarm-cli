package label

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "key=value format",
			input:     "team=frontend",
			wantKey:   "team",
			wantValue: "frontend",
			wantErr:   false,
		},
		{
			name:      "key only format",
			input:     "team",
			wantKey:   "team",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:      "key with dots",
			input:     "app.kubernetes.io/name=myapp",
			wantKey:   "app.kubernetes.io/name",
			wantValue: "myapp",
			wantErr:   false,
		},
		{
			name:      "value with slashes",
			input:     "path=foo/bar/baz",
			wantKey:   "path",
			wantValue: "foo/bar/baz",
			wantErr:   false,
		},
		{
			name:      "value with hyphens and underscores",
			input:     "ticket=PROJ-123_abc",
			wantKey:   "ticket",
			wantValue: "PROJ-123_abc",
			wantErr:   false,
		},
		{
			name:      "empty key",
			input:     "",
			wantKey:   "",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "empty key with value",
			input:     "=value",
			wantKey:   "",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "key starting with number",
			input:     "1team=frontend",
			wantKey:   "",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "reserved prefix",
			input:     "swarm.internal=value",
			wantKey:   "",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "key with special chars",
			input:     "team@work=value",
			wantKey:   "",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "value with special chars",
			input:     "team=val@ue",
			wantKey:   "",
			wantValue: "",
			wantErr:   true,
		},
		{
			name:      "empty value",
			input:     "team=",
			wantKey:   "team",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:      "value with equals sign",
			input:     "equation=a=b",
			wantKey:   "equation",
			wantValue: "a=b",
			wantErr:   true, // = not allowed in value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if key != tt.wantKey {
					t.Errorf("Parse() key = %v, want %v", key, tt.wantKey)
				}
				if value != tt.wantValue {
					t.Errorf("Parse() value = %v, want %v", value, tt.wantValue)
				}
			}
		})
	}
}

func TestParseMultiple(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   []string{},
			want:    nil,
			wantErr: false,
		},
		{
			name:  "single label",
			input: []string{"team=frontend"},
			want:  map[string]string{"team": "frontend"},
		},
		{
			name:  "multiple labels",
			input: []string{"team=frontend", "priority=high", "env=staging"},
			want:  map[string]string{"team": "frontend", "priority": "high", "env": "staging"},
		},
		{
			name:  "duplicate keys (later wins)",
			input: []string{"team=a", "team=b"},
			want:  map[string]string{"team": "b"},
		},
		{
			name:    "invalid label",
			input:   []string{"team=frontend", "swarm.internal=value"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMultiple(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMultiple() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("ParseMultiple() len = %v, want %v", len(got), len(tt.want))
					return
				}
				for k, v := range tt.want {
					if got[k] != v {
						t.Errorf("ParseMultiple()[%s] = %v, want %v", k, got[k], v)
					}
				}
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name         string
		agentLabels  map[string]string
		filterLabels map[string]string
		want         bool
	}{
		{
			name:         "empty filter matches everything",
			agentLabels:  map[string]string{"team": "frontend"},
			filterLabels: nil,
			want:         true,
		},
		{
			name:         "empty filter matches empty agent",
			agentLabels:  nil,
			filterLabels: nil,
			want:         true,
		},
		{
			name:         "filter on empty agent labels",
			agentLabels:  nil,
			filterLabels: map[string]string{"team": "frontend"},
			want:         false,
		},
		{
			name:         "exact match",
			agentLabels:  map[string]string{"team": "frontend", "priority": "high"},
			filterLabels: map[string]string{"team": "frontend"},
			want:         true,
		},
		{
			name:         "multiple filter match",
			agentLabels:  map[string]string{"team": "frontend", "priority": "high", "env": "staging"},
			filterLabels: map[string]string{"team": "frontend", "priority": "high"},
			want:         true,
		},
		{
			name:         "value mismatch",
			agentLabels:  map[string]string{"team": "frontend"},
			filterLabels: map[string]string{"team": "backend"},
			want:         false,
		},
		{
			name:         "key existence check (empty filter value)",
			agentLabels:  map[string]string{"team": "frontend"},
			filterLabels: map[string]string{"team": ""},
			want:         true,
		},
		{
			name:         "key missing",
			agentLabels:  map[string]string{"team": "frontend"},
			filterLabels: map[string]string{"priority": ""},
			want:         false,
		},
		{
			name:         "partial match fails",
			agentLabels:  map[string]string{"team": "frontend"},
			filterLabels: map[string]string{"team": "frontend", "priority": "high"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.agentLabels, tt.filterLabels)
			if got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		labels map[string]string
		want   string
	}{
		{
			name:   "nil labels",
			labels: nil,
			want:   "-",
		},
		{
			name:   "empty labels",
			labels: map[string]string{},
			want:   "-",
		},
		{
			name:   "single label",
			labels: map[string]string{"team": "frontend"},
			want:   "team=frontend",
		},
		{
			name:   "multiple labels (sorted)",
			labels: map[string]string{"team": "frontend", "env": "staging", "priority": "high"},
			want:   "env=staging,priority=high,team=frontend",
		},
		{
			name:   "label without value",
			labels: map[string]string{"team": ""},
			want:   "team",
		},
		{
			name:   "mixed labels",
			labels: map[string]string{"team": "frontend", "urgent": ""},
			want:   "team=frontend,urgent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format(tt.labels)
			if got != tt.want {
				t.Errorf("Format() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]string
		override map[string]string
		want     map[string]string
	}{
		{
			name:     "both nil",
			base:     nil,
			override: nil,
			want:     nil,
		},
		{
			name:     "nil base",
			base:     nil,
			override: map[string]string{"team": "frontend"},
			want:     map[string]string{"team": "frontend"},
		},
		{
			name:     "nil override",
			base:     map[string]string{"team": "frontend"},
			override: nil,
			want:     map[string]string{"team": "frontend"},
		},
		{
			name:     "no overlap",
			base:     map[string]string{"team": "frontend"},
			override: map[string]string{"priority": "high"},
			want:     map[string]string{"team": "frontend", "priority": "high"},
		},
		{
			name:     "override takes precedence",
			base:     map[string]string{"team": "frontend", "priority": "low"},
			override: map[string]string{"priority": "high"},
			want:     map[string]string{"team": "frontend", "priority": "high"},
		},
		{
			name:     "both empty",
			base:     map[string]string{},
			override: map[string]string{},
			want:     nil, // Empty result becomes nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.base, tt.override)
			if len(got) != len(tt.want) {
				t.Errorf("Merge() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("Merge()[%s] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
