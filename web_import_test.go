package main

import (
	"testing"
)

func TestConvertToRawURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub File URL",
			input:    "https://github.com/phoeagon/RectangleWinPlus/blob/main/config.example.yaml",
			expected: "https://raw.githubusercontent.com/phoeagon/RectangleWinPlus/main/config.example.yaml",
		},
		{
			name:     "GitHub Gist URL",
			input:    "https://gist.github.com/phoeagon/9135e2ecec336384fc40c16dee22a959",
			expected: "https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959/raw",
		},
		{
			name:     "GitHub Gist URL with Fragment",
			input:    "https://gist.github.com/phoeagon/9135e2ecec336384fc40c16dee22a959#file-config-yaml",
			expected: "https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959/raw",
		},
		{
			name:     "GitHub Gist Raw Domain without /raw",
			input:    "https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959",
			expected: "https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959/raw",
		},
		{
			name:     "GitHub Gist Raw Domain with /raw",
			input:    "https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959/raw",
			expected: "https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959/raw",
		},
		{
			name:     "Bitbucket URL",
			input:    "https://bitbucket.org/user/repo/src/master/config.yaml",
			expected: "https://bitbucket.org/user/repo/raw/master/config.yaml",
		},
		{
			name:     "Already Raw GitHub URL",
			input:    "https://raw.githubusercontent.com/phoeagon/RectangleWinPlus/main/config.example.yaml",
			expected: "https://raw.githubusercontent.com/phoeagon/RectangleWinPlus/main/config.example.yaml",
		},
		{
			name:     "Generic URL (No Change)",
			input:    "https://example.com/config.yaml",
			expected: "https://example.com/config.yaml",
		},
		{
			name:     "URL with Fragment (Strip Fragment)",
			input:    "https://example.com/config.yaml#section1",
			expected: "https://example.com/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToRawURL(tt.input)
			if result != tt.expected {
				t.Errorf("convertToRawURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
