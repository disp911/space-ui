package sub

import (
	"strings"
	"testing"
)

// Links and the JSON subscription give the client the inbound's SpiderX, a
// page of the site; only an inbound without one gets a random path.
func TestSpiderXIsTheInbounds(t *testing.T) {
	reality := func(spx any) map[string]any {
		settings := map[string]any{"publicKey": "k", "fingerprint": "chrome"}
		if spx != nil {
			settings["spiderX"] = spx
		}
		return map[string]any{"serverNames": []any{"docs.example.com"}, "shortIds": []any{"ab"}, "settings": settings}
	}
	random := func(p string) bool { return strings.HasPrefix(p, "/") && len(p) == 16 }

	for _, tc := range []struct {
		spx    any
		want   string
		random bool
	}{
		{"/docs/", "/docs/", false},
		{" / ", "/", false},
		{"", "", true},
		{"  ", "", true},
		{nil, "", true},
	} {
		params := map[string]string{}
		applyShareRealityParams(map[string]any{"realitySettings": reality(tc.spx)}, params)
		json := (&SubJsonService{}).realityData(reality(tc.spx))["spiderX"].(string)
		for name, got := range map[string]string{"link": params["spx"], "json": json} {
			if tc.random && !random(got) || !tc.random && got != tc.want {
				t.Errorf("%s with spiderX %q: %q", name, tc.spx, got)
			}
		}
	}
}
