package ec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsECV3Version(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"3.0.0+k8s-1.34", true},
		{"v3.0.0+k8s-1.34", true},
		{"3.1.2-beta+k8s-1.35", true},
		{"v3.9.9", true},
		{"2.9.9+k8s-1.29", false},
		{"v2.0.0+k8s-1.29", false},
		{"v1.2.2+k8s-1.29", false},
		{"30.0.0", false},
		{"v30.0.0", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, IsECV3Version(tt.version))
		})
	}
}
