package main

import (
	"strings"
)

// convertToRawURL converts various URL formats to their raw content equivalents
// example urls:
// https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959
// https://gist.githubusercontent.com/phoeagon/9135e2ecec336384fc40c16dee22a959/raw
// https://gist.github.com/phoeagon/9135e2ecec336384fc40c16dee22a959#file-config-yaml
// https://raw.githubusercontent.com/phoeagon/RectangleWinPlus/refs/heads/main/config.example.yaml
// https://github.com/phoeagon/RectangleWinPlus/blob/main/config.example.yaml
func convertToRawURL(url string) string {
	// Strip any fragment (e.g. #file-config-yaml)
	if idx := strings.Index(url, "#"); idx != -1 {
		url = url[:idx]
	}

	// GitHub file URLs: github.com/user/repo/blob/branch/file -> raw.githubusercontent.com/user/repo/branch/file
	if strings.Contains(url, "github.com") && strings.Contains(url, "/blob/") {
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		url = strings.Replace(url, "/blob/", "/", 1)
		return url
	}

	// GitHub Gist URLs: gist.github.com/user/gist_id -> gist.githubusercontent.com/user/gist_id/raw
	// Note: This will fetch the first file in the gist. For multi-file gists, users should use the raw URL directly.
	if strings.Contains(url, "gist.github.com") && !strings.Contains(url, "githubusercontent.com") {
		url = strings.Replace(url, "gist.github.com", "gist.githubusercontent.com", 1)
		// Remove trailing slash if present
		url = strings.TrimSuffix(url, "/")
		// Add /raw to get the raw content of the first file
		url = url + "/raw"
		return url
	}

	// Handle gist.githubusercontent.com URLs that are missing /raw
	if strings.Contains(url, "gist.githubusercontent.com") && !strings.Contains(url, "/raw") {
		url = strings.TrimSuffix(url, "/")
		url = url + "/raw"
		return url
	}

	// Bitbucket URLs: bitbucket.org/user/repo/src/branch/file -> bitbucket.org/user/repo/raw/branch/file
	if strings.Contains(url, "bitbucket.org") && strings.Contains(url, "/src/") {
		url = strings.Replace(url, "/src/", "/raw/", 1)
		return url
	}

	// Already raw or other URLs - return as-is
	return url
}
