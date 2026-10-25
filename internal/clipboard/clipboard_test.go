package clipboard

import (
	"errors"
	"testing"

	"github.com/zyedidia/clipper"
)

func TestWaylandCopyArgsForceTextPlain(t *testing.T) {
	tests := []struct {
		name string
		reg  string
		want []string
	}{
		{
			name: "clipboard",
			reg:  clipper.RegClipboard,
			want: []string{"--type", "text/plain"},
		},
		{
			name: "primary",
			reg:  clipper.RegPrimary,
			want: []string{"--type", "text/plain", "--primary"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := waylandCopyArgs(tt.reg)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("args = %#v, want %#v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("args = %#v, want %#v", got, tt.want)
				}
			}
		})
	}
}

func TestWaylandCopyArgsRejectInvalidRegister(t *testing.T) {
	_, err := waylandCopyArgs("invalid")
	var invalid *clipper.ErrInvalidReg
	if !errors.As(err, &invalid) {
		t.Fatalf("err = %v, want ErrInvalidReg", err)
	}
}
