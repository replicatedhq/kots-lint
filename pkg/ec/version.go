package ec

import "strings"

// IsECV3Version reports whether version represents an Embedded Cluster v3 release.
// It accepts versions with or without a leading "v" (e.g. "3.0.0+k8s-1.34" or "v3.0.0+k8s-1.34").
func IsECV3Version(version string) bool {
	return strings.HasPrefix(version, "3.") || strings.HasPrefix(version, "v3.")
}
