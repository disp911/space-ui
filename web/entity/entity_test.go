package entity

import "testing"

func TestListsSettingsAreChecked(t *testing.T) {
	valid := func() *AllSetting {
		return &AllSetting{
			WebPort: 2053, SubPort: 2096, WebBasePath: "/panel/", SubPath: "/sub/", SubJsonPath: "/json/", SubClashPath: "/clash/",
			TimeLocation: "UTC", ListsPath: "lists", ListsSecret: "abcdefghijKLMNOP1234",
		}
	}
	s := valid()
	if err := s.CheckValid(); err != nil {
		t.Fatal(err)
	}
	if s.ListsPath != "/lists/" {
		t.Fatalf("path %q", s.ListsPath)
	}
	for name, change := range map[string]func(*AllSetting){
		"the root":              func(s *AllSetting) { s.ListsPath = "/" },
		"the panel's path":      func(s *AllSetting) { s.ListsPath = "/panel/" },
		"under the panel's":     func(s *AllSetting) { s.ListsPath = "/panel/lists/" },
		"over the subscription": func(s *AllSetting) { s.ListsPath = "/"; s.SubPath = "/sub/" },
		"the subscription's":    func(s *AllSetting) { s.ListsPath = "/sub/" },
		"a short secret":        func(s *AllSetting) { s.ListsSecret = "short" },
		"a secret with a slash": func(s *AllSetting) { s.ListsSecret = "abcdefghij/KLMNOP1234" },
		"no secret":             func(s *AllSetting) { s.ListsSecret = "" },
	} {
		s := valid()
		change(s)
		if err := s.CheckValid(); err == nil {
			t.Errorf("%s passes", name)
		}
	}
	s = valid()
	s.WebBasePath = "/"
	if err := s.CheckValid(); err != nil {
		t.Fatalf("lists beside a panel at the root: %v", err)
	}
}
