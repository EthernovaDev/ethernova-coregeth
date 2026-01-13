package eth

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestParseNodeVersion(t *testing.T) {
	tests := []struct {
		name   string
		want   semver
		wantOK bool
	}{
		{
			name:   "CoreGeth/v1.2.6/windows-amd64/go1.20",
			want:   semver{major: 1, minor: 2, patch: 6},
			wantOK: true,
		},
		{
			name:   "CoreGeth/v1.2.6-abcdef/windows-amd64/go1.20",
			want:   semver{major: 1, minor: 2, patch: 6},
			wantOK: true,
		},
		{
			name:   "CoreGeth/invalid",
			wantOK: false,
		},
	}
	for _, test := range tests {
		got, ok := parseNodeVersion(test.name)
		if ok != test.wantOK {
			t.Fatalf("parseNodeVersion(%q) ok=%v want=%v", test.name, ok, test.wantOK)
		}
		if ok && got != test.want {
			t.Fatalf("parseNodeVersion(%q)=%v want=%v", test.name, got, test.want)
		}
	}
}

func TestEnforcePeerVersion(t *testing.T) {
	min := minPeerVersion()
	minName := fmt.Sprintf("%s/v%d.%d.%d/windows-amd64/go1.20", params.VersionName, min.major, min.minor, min.patch)
	if err := enforcePeerVersion(minName); err != nil {
		t.Fatalf("expected min version to pass, got %v", err)
	}

	old := min
	if old.patch > 0 {
		old.patch--
	} else if old.minor > 0 {
		old.minor--
		old.patch = 0
	} else if old.major > 0 {
		old.major--
		old.minor = 0
		old.patch = 0
	}
	oldName := fmt.Sprintf("%s/v%d.%d.%d/windows-amd64/go1.20", params.VersionName, old.major, old.minor, old.patch)
	if err := enforcePeerVersion(oldName); err == nil {
		t.Fatalf("expected old version to be rejected")
	}

	if err := enforcePeerVersion("OtherClient/v9.9.9/linux-amd64/go1.20"); err == nil {
		t.Fatalf("expected unsupported client to be rejected")
	}
}
