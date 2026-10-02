// Package firefox reads Firefox installs and profiles: profiles.ini, installs.ini, compatibility.ini and profile locks.
package firefox

import (
	"encoding/binary"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/headwalluk/foxtrainer/internal/cityhash"
)

// InstallHash returns the hash Firefox uses in [Install<hash>] for the directory holding its executable.
//
// The directory must already be symlink-resolved, with no trailing separator. The result is
// uppercase hex without zero-padding, so roughly 1 in 16 hashes has 15 digits.
func InstallHash(installDir string) string {
	codeUnits := utf16.Encode([]rune(installDir))

	encoded := make([]byte, 0, len(codeUnits)*2)
	for _, codeUnit := range codeUnits {
		encoded = binary.LittleEndian.AppendUint16(encoded, codeUnit)
	}

	return strings.ToUpper(strconv.FormatUint(cityhash.Hash64(encoded), 16))
}
