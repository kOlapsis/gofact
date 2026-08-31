//go:build linux

package facturx

func browserCandidates() []string {
	return []string{
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/opt/google/chrome/chrome",
		"/usr/bin/microsoft-edge",
		"/usr/bin/microsoft-edge-stable",
		"/usr/bin/brave-browser",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	}
}

func browserCommands() []string {
	return []string{
		"google-chrome", "google-chrome-stable", "microsoft-edge",
		"brave-browser", "chromium", "chromium-browser",
	}
}
