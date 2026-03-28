package simpleapi

import "testing"

func TestNormalizeInviteLoginURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty falls back to deployed host",
			in:   "",
			want: "https://backatage.stepbycode.work/login",
		},
		{
			name: "legacy host is rewritten",
			in:   "https://backstage.stepbycode.work/login",
			want: "https://backatage.stepbycode.work/login",
		},
		{
			name: "legacy host preserves query",
			in:   "https://backstage.stepbycode.work/login?next=%2Finvite",
			want: "https://backatage.stepbycode.work/login?next=%2Finvite",
		},
		{
			name: "already corrected host stays unchanged",
			in:   "https://backatage.stepbycode.work/login",
			want: "https://backatage.stepbycode.work/login",
		},
		{
			name: "other hosts stay unchanged",
			in:   "https://example.com/login",
			want: "https://example.com/login",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeInviteLoginURL(tt.in); got != tt.want {
				t.Fatalf("normalizeInviteLoginURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
