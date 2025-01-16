package pin

import (
	"testing"
)

func Test_isImmutableAction(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   bool
	}{
		{
			name:   "immutable action - 1",
			action: "actions/checkout@v4.2.2",
			want:   true,
		},
		{
			name:   "immutable action - 2",
			action: "step-security/wait-for-secrets@v1.2.0",
			want:   true,
		},
		{
			name:   "non immutable action(valid action)",
			action: "sailikhith-stepsecurity/hello-action@v1.0.2",
			want:   false,
		},
		{
			name:   "non immutable action(invalid action)",
			action: "sailikhith-stepsecurity/no-such-action@v1.0.2",
			want:   false,
		},
		{
			name:   " action with release tag doesn't exist",
			action: "actions/checkout@1.2.3",
			want:   false,
		},
		{
			name:   "invalid action format",
			action: "invalid-format",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := IsImmutableAction(tt.action)
			if got != tt.want {
				t.Errorf("isImmutableAction() = %v, want %v", got, tt.want)
			}
		})
	}
}
