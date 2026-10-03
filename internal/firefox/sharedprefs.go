package firefox

import (
	"slices"
	"strings"
)

// groupSharedPrefNames are Firefox 158's permanentSharedPrefs: kept in the Profile Group store, which overrides user.js.
var groupSharedPrefNames = []string{
	"app.shield.optoutstudies.enabled",
	"browser.backup.enabled_on.profiles",
	"browser.crashReports.unsubmittedCheck.autoSubmit2",
	"browser.discovery.enabled",
	"browser.shell.checkDefaultBrowser",
	"datareporting.dau.cachedUsageProfileGroupID",
	"datareporting.healthreport.uploadEnabled",
	"datareporting.usage.uploadEnabled",
	"toolkit.telemetry.cachedProfileGroupID",
}

// groupSharedPrefPrefixes are the permanentSharedPrefs entries that cover a whole branch.
var groupSharedPrefPrefixes = []string{
	"browser.shell.customIcon.",
	"datareporting.policy.",
	"termsofuse.",
}

// IsGroupSharedPref reports whether Firefox keeps name in the Profile Group store for grouped profiles.
func IsGroupSharedPref(name string) bool {
	return slices.Contains(groupSharedPrefNames, name) ||
		slices.ContainsFunc(groupSharedPrefPrefixes, func(prefix string) bool { return strings.HasPrefix(name, prefix) })
}
