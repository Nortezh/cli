package selfupdate

import "testing"

func TestNewer(t *testing.T) {
	cases := []struct {
		current, remote string
		want            bool
	}{
		{"0.6.1", "v0.7.0", true},
		{"0.6.1", "v0.6.2", true},
		{"0.6.1", "v0.6.1", false},
		{"0.7.0", "v0.6.1", false},
		{"1.0.0", "v0.9.9", false},
		{"v0.6.1", "v0.7.0", true},
		{"dev", "v0.7.0", true},     // unparseable current -> always older
		{"0.6.1", "garbage", false}, // unparseable remote -> never newer
		{"0.6.1-rc1", "v0.6.1", false},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.remote); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.current, c.remote, got, c.want)
		}
	}
}

func TestArchiveName(t *testing.T) {
	cases := []struct {
		tag, goos, goarch string
		wantName          string
		wantZip           bool
	}{
		{"v0.7.0", "darwin", "arm64", "ntzh_0.7.0_Darwin_arm64.tar.gz", false},
		{"v0.7.0", "darwin", "amd64", "ntzh_0.7.0_Darwin_x86_64.tar.gz", false},
		{"v0.7.0", "linux", "amd64", "ntzh_0.7.0_Linux_x86_64.tar.gz", false},
		{"0.7.0", "linux", "arm64", "ntzh_0.7.0_Linux_arm64.tar.gz", false},
		{"v0.7.0", "windows", "amd64", "ntzh_0.7.0_Windows_x86_64.zip", true},
	}
	for _, c := range cases {
		name, isZip := archiveName(c.tag, c.goos, c.goarch)
		if name != c.wantName || isZip != c.wantZip {
			t.Errorf("archiveName(%q,%q,%q) = (%q,%v), want (%q,%v)",
				c.tag, c.goos, c.goarch, name, isZip, c.wantName, c.wantZip)
		}
	}
}
