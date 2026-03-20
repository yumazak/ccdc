package proxy

import "testing"

func TestBuildNoProxy(t *testing.T) {
	tests := []struct {
		name       string
		withDocker bool
		withJoy    bool
		want       string
	}{
		{
			name:       "both false",
			withDocker: false,
			withJoy:    false,
			want:       "localhost,127.0.0.1,proxy",
		},
		{
			name:       "docker only",
			withDocker: true,
			withJoy:    false,
			want:       "localhost,127.0.0.1,proxy,socket-proxy",
		},
		{
			name:       "joy only",
			withDocker: false,
			withJoy:    true,
			want:       "localhost,127.0.0.1,proxy,joy-proxy",
		},
		{
			name:       "both true",
			withDocker: true,
			withJoy:    true,
			want:       "localhost,127.0.0.1,proxy,socket-proxy,joy-proxy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNoProxy(tt.withDocker, tt.withJoy)
			if got != tt.want {
				t.Errorf("buildNoProxy(%v, %v) = %q, want %q", tt.withDocker, tt.withJoy, got, tt.want)
			}
		})
	}
}
