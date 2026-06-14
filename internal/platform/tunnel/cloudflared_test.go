package tunnel

import "testing"

func TestExtractQuickTunnelURL(t *testing.T) {
	tests := []struct {
		name    string
		stderr  string
		wantURL string
		wantOK  bool
	}{
		{
			name: "realistic quick-tunnel banner",
			stderr: `2024-01-01T00:00:00Z INF Thank you for trying Cloudflare Tunnel.
2024-01-01T00:00:00Z INF Requesting new quick Tunnel on trycloudflare.com...
2024-01-01T00:00:00Z INF +--------------------------------------------------------------------------------------------+
2024-01-01T00:00:00Z INF |  Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):  |
2024-01-01T00:00:00Z INF |  https://brave-tree-1234.trycloudflare.com                                                 |
2024-01-01T00:00:00Z INF +--------------------------------------------------------------------------------------------+`,
			wantURL: "https://brave-tree-1234.trycloudflare.com",
			wantOK:  true,
		},
		{
			name:    "no url present",
			stderr:  "2024-01-01T00:00:00Z INF Starting tunnel\n2024-01-01T00:00:00Z ERR connection failed",
			wantURL: "",
			wantOK:  false,
		},
		{
			name: "multiple urls returns first",
			stderr: `https://first-one-0001.trycloudflare.com
https://second-two-0002.trycloudflare.com`,
			wantURL: "https://first-one-0001.trycloudflare.com",
			wantOK:  true,
		},
		{
			name:    "empty input",
			stderr:  "",
			wantURL: "",
			wantOK:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotOK := extractQuickTunnelURL(tt.stderr)
			if gotURL != tt.wantURL || gotOK != tt.wantOK {
				t.Fatalf("extractQuickTunnelURL() = (%q, %v), want (%q, %v)", gotURL, gotOK, tt.wantURL, tt.wantOK)
			}
		})
	}
}

func TestIsValidExecutable(t *testing.T) {
	elf := []byte{0x7f, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}
	machO := []byte{0xcf, 0xfa, 0xed, 0xfe, 0x07, 0x00, 0x00, 0x01}
	pe := []byte{'M', 'Z', 0x90, 0x00, 0x03, 0x00, 0x00, 0x00}
	html := []byte("<!DOCTYPE html><html>not a binary</html>")

	tests := []struct {
		name string
		head []byte
		goos string
		want bool
	}{
		{"linux elf ok", elf, "linux", true},
		{"darwin mach-o ok", machO, "darwin", true},
		{"windows pe ok", pe, "windows", true},
		{"linux rejects html", html, "linux", false},
		{"darwin rejects elf", elf, "darwin", false},
		{"windows rejects elf", elf, "windows", false},
		{"too short", []byte{0x7f}, "linux", false},
		{"empty", nil, "linux", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidExecutable(tt.head, tt.goos); got != tt.want {
				t.Fatalf("isValidExecutable(%v, %q) = %v, want %v", tt.head, tt.goos, got, tt.want)
			}
		})
	}
}
