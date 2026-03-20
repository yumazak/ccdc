package proxy

import "testing"

func TestBuildNoProxy(t *testing.T) {
	tests := []struct {
		name    string
		withJoy bool
		want    string
	}{
		{
			name:    "without joy",
			withJoy: false,
			want:    "localhost,127.0.0.1,proxy",
		},
		{
			name:    "with joy",
			withJoy: true,
			want:    "localhost,127.0.0.1,proxy,joy-proxy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNoProxy(tt.withJoy)
			if got != tt.want {
				t.Errorf("buildNoProxy(%v) = %q, want %q", tt.withJoy, got, tt.want)
			}
		})
	}
}
