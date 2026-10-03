package firefox

import "testing"

func TestIsGroupSharedPref(test *testing.T) {
	expectations := map[string]bool{
		"datareporting.healthreport.uploadEnabled":   true,
		"datareporting.policy.dataSubmissionEnabled": true,
		"termsofuse.acceptedDate":                    true,
		"browser.shell.checkDefaultBrowser":          true,
		"browser.shell.customIcon.path":              true,
		"browser.shell.didSkipDefaultBrowserCheck":   false,
		"datareporting.healthreport.service.enabled": false,
		"browser.ml.enable":                          false,
	}

	for name, expected := range expectations {
		if found := IsGroupSharedPref(name); found != expected {
			test.Errorf("IsGroupSharedPref(%q) = %v, expected %v", name, found, expected)
		}
	}
}
