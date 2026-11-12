package web

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/disp911/space-ui/v3/logger"
	"github.com/disp911/space-ui/v3/web/locale"

	"github.com/op/go-logging"
)

// TestServerMessagesAreTranslated checks that every message the server
// translates (I18nWeb(c, "key") in the Go code) is in the translation files.
// The files also feed the frontend, whose own tables have their own check.
func TestServerMessagesAreTranslated(t *testing.T) {
	logger.InitLogger(logging.CRITICAL)
	if err := locale.InitLocalizer(i18nFS); err != nil {
		t.Fatal(err)
	}
	call := regexp.MustCompile(`I18nWeb\(c, "([A-Za-z0-9_.]+)"`)
	keys := map[string]string{}
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".")) && path != ".." {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			keys[m[1]] = path
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) < 50 {
		t.Fatalf("found only %d messages; is the pattern still right?", len(keys))
	}
	for key, file := range keys {
		if got := locale.I18n(locale.Web, key); got == "" || got == key {
			t.Errorf("%s: %q is not translated", file, key)
		}
	}
}
