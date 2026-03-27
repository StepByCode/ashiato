package corsutil

import "testing"

func TestIsOriginAllowed(t *testing.T) {
	t.Parallel()

	allowed := BuildAllowedOrigins([]string{
		"https://*.stepbycode.work",
		"*.vercel.app",
		"https://api.example.com",
	})

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "default production origin", origin: "https://backatage.stepbycode.work", want: true},
		{name: "https wildcard pattern", origin: "https://preview.stepbycode.work", want: true},
		{name: "bare wildcard pattern", origin: "https://ashiato-git-main-stepbycode.vercel.app", want: true},
		{name: "exact configured origin", origin: "https://api.example.com", want: true},
		{name: "default localhost", origin: "http://localhost:3000", want: true},
		{name: "wrong scheme", origin: "http://preview.stepbycode.work", want: false},
		{name: "wrong domain", origin: "https://example.com", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsOriginAllowed(tt.origin, allowed); got != tt.want {
				t.Fatalf("IsOriginAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}
