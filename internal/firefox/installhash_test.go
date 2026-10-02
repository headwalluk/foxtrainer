package firefox

import "testing"

func TestInstallHashKnownValues(test *testing.T) {
	cases := map[string]string{
		"/usr/lib/firefox-devedition":              "BCAEFFD141225C21", // Mozilla apt Developer Edition
		"/opt/firefox":                             "6AFDA46A1A8AD48",  // 15 digits: leading zero nibble dropped
		"/usr/lib/firefox-esr":                     "3B6073811A6ABF12", // Debian ESR
		"/usr/lib/firefox":                         "4F96D1932A9F858E", // Ubuntu/Debian deb release
		`C:\Program Files\Mozilla Firefox`:         "308046B0AF4A39CB", // Windows installer default
		"/Applications/Firefox.app/Contents/MacOS": "2656FF1E876E9973", // macOS default
	}

	for installDir, want := range cases {
		if got := InstallHash(installDir); got != want {
			test.Errorf("%s: got %s, want %s", installDir, got, want)
		}
	}
}
