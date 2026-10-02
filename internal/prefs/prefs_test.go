package prefs

import (
	"math"
	"testing"
)

func TestParseStatement(test *testing.T) {
	cases := []struct {
		input    string
		want     Statement
		wantFail bool
	}{
		{input: `user_pref("browser.ml.enable", false);`, want: Statement{Function: "user_pref", Name: "browser.ml.enable", Value: Bool(false)}},
		{input: `user_pref("network.http.max-connections", 1800); // default=900`, want: Statement{
			Function: "user_pref", Name: "network.http.max-connections", Value: Int(1800), Trailing: "default=900",
		}},
		{input: `user_pref("geo.provider.network.url", "https://beacondb.net/v1/geolocate");`, want: Statement{
			Function: "user_pref", Name: "geo.provider.network.url", Value: String("https://beacondb.net/v1/geolocate"),
		}},
		{input: `user_pref("browser.sessionstore.restore_on_demand", -1);`, want: Statement{
			Function: "user_pref", Name: "browser.sessionstore.restore_on_demand", Value: Int(-1),
		}},
		{input: `user_pref("network.trr.resolvers", '[{ "name": "x" }]');`, want: Statement{
			Function: "user_pref", Name: "network.trr.resolvers", Value: String(`[{ "name": "x" }]`),
		}},
		{input: `user_pref("x.escaped", "{\"screen\":\"\",\"complete\":true}"); // ; ) "quotes" in comment`, want: Statement{
			Function: "user_pref", Name: "x.escaped", Value: String(`{"screen":"","complete":true}`), Trailing: `; ) "quotes" in comment`,
		}},
		{input: `pref("app.update.channel", "aurora");`, want: Statement{Function: "pref", Name: "app.update.channel", Value: String("aurora")}},
		{input: `user_pref("pdfjs.defaultZoomValue", page-width);`, wantFail: true},
		{input: `user_pref("browser.contentanalysis.default_result", 0; // missing paren`, wantFail: true},
		{input: `(alternate) user_pref("dom.popup_allowed_events", "click");`, wantFail: true},
		{input: `user_pref("too.big", 3000000000);`, wantFail: true},
	}

	for _, testCase := range cases {
		got, parseError := ParseStatement(testCase.input)

		switch {
		case testCase.wantFail && parseError == nil:
			test.Errorf("%s: want an error, got %+v", testCase.input, got)
		case !testCase.wantFail && parseError != nil:
			test.Errorf("%s: unexpected error %v", testCase.input, parseError)
		case !testCase.wantFail && got != testCase.want:
			test.Errorf("%s:\n got %+v\nwant %+v", testCase.input, got, testCase.want)
		}
	}
}

func TestFormatUserPrefRoundTrips(test *testing.T) {
	values := []Value{Bool(true), Int(-5), Int(math.MaxInt32), String(`quote " backslash \ newline` + "\n" + `unicode £`)}

	for _, value := range values {
		line := FormatUserPref("test.pref", value)

		parsed, parseError := ParseStatement(line)
		if parseError != nil || parsed.Value != value || parsed.Name != "test.pref" {
			test.Errorf("%s: got %+v, %v", line, parsed, parseError)
		}
	}
}

func TestFromTOML(test *testing.T) {
	good := map[any]Value{true: Bool(true), int64(42): Int(42), "blocked": String("blocked")}

	for raw, want := range good {
		if got, convertError := FromTOML(raw); convertError != nil || got != want {
			test.Errorf("%v: got %+v, %v", raw, got, convertError)
		}
	}

	for _, raw := range []any{1.5, int64(math.MaxInt32) + 1, []any{"x"}} {
		if _, convertError := FromTOML(raw); convertError == nil {
			test.Errorf("%v: want an error", raw)
		}
	}
}
