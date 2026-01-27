package cmd

import (
	"os"
	"testing"
)

func TestResolveEditor(t *testing.T) {
	tests := []struct {
		name     string
		override string
		visual   string
		editor   string
		want     string
	}{
		{
			name:     "override takes precedence",
			override: "custom-editor",
			visual:   "visual-editor",
			editor:   "default-editor",
			want:     "custom-editor",
		},
		{
			name:   "VISUAL over EDITOR",
			visual: "visual-editor",
			editor: "default-editor",
			want:   "visual-editor",
		},
		{
			name:   "EDITOR as fallback",
			editor: "default-editor",
			want:   "default-editor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldVisual := os.Getenv("VISUAL")
			oldEditor := os.Getenv("EDITOR")
			defer func() {
				os.Setenv("VISUAL", oldVisual)
				os.Setenv("EDITOR", oldEditor)
			}()

			os.Setenv("VISUAL", tt.visual)
			os.Setenv("EDITOR", tt.editor)

			got := resolveEditor(tt.override)
			// Note: on systems without vim/vi/nano, fallback won't work
			// so we only check when we expect a specific result
			if tt.want != "" && got != tt.want {
				t.Errorf("resolveEditor() = %v, want %v", got, tt.want)
			}
		})
	}
}
