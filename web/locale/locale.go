// Package locale provides internationalization (i18n) support for the Space UI web panel,
// including translation loading, localization, and middleware for the web interface.
package locale

import (
	"embed"
	"io/fs"
	"os"
	"strings"
	"sync"

	"github.com/disp911/space-ui/v3/logger"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

var (
	bundleMu   sync.Mutex
	i18nBundle *i18n.Bundle
)

// I18nType represents the type of interface for internationalization.
type I18nType string

const (
	Web I18nType = "web" // Web interface type
)

// newBundle returns an empty bundle with English as the default language.
func newBundle() *i18n.Bundle {
	bundle := i18n.NewBundle(language.MustParse("en-US"))
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	return bundle
}

// InitLocalizer initializes the internationalization system with embedded translation files.
func InitLocalizer(i18nFS embed.FS) error {
	bundle := newBundle()
	if err := parseTranslationFiles(i18nFS, bundle); err != nil {
		return err
	}
	bundleMu.Lock()
	i18nBundle = bundle
	bundleMu.Unlock()
	return nil
}

// currentBundle returns the loaded translations. The subscription server can
// take requests before the panel has loaded them; it then reads them from
// disk.
func currentBundle() *i18n.Bundle {
	bundleMu.Lock()
	defer bundleMu.Unlock()
	if i18nBundle == nil {
		i18nBundle = newBundle()
		if err := loadTranslationsFromDisk(i18nBundle); err != nil {
			logger.Warning("i18n lazy load failed:", err)
		}
	}
	return i18nBundle
}

// createTemplateData creates a template data map from parameters with optional separator.
func createTemplateData(params []string, separator ...string) map[string]any {
	var sep string = "=="
	if len(separator) > 0 {
		sep = separator[0]
	}

	templateData := make(map[string]any)
	for _, param := range params {
		parts := strings.SplitN(param, sep, 2)
		templateData[parts[0]] = parts[1]
	}

	return templateData
}

// localize returns the message for key in the localizer's language, or an
// empty string if there is none.
func localize(localizer *i18n.Localizer, key string, params ...string) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: createTemplateData(params),
	})
	if err != nil {
		logger.Errorf("Failed to localize message: %v", err)
		return ""
	}
	return msg
}

// I18n retrieves the English message for the given key and type, with
// optional template parameters. Pages and handlers are translated into the
// language of their request instead; this serves where there is none.
func I18n(i18nType I18nType, key string, params ...string) string {
	if i18nType != Web {
		logger.Errorf("Invalid type for I18n: %s", i18nType)
		return ""
	}
	return localize(i18n.NewLocalizer(currentBundle(), "en-US"), key, params...)
}

// LocalizerMiddleware returns a Gin middleware that sets up localization for web requests.
// It determines the user's language from cookies or Accept-Language header,
// creates a localizer instance, and stores it in the Gin context for use in handlers.
// Also provides the I18n function in the context for handlers.
func LocalizerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var lang string

		if cookie, err := c.Request.Cookie("lang"); err == nil {
			lang = cookie.Value
		} else {
			lang = c.GetHeader("Accept-Language")
		}

		localizer := i18n.NewLocalizer(currentBundle(), lang)
		c.Set("I18n", func(i18nType I18nType, key string, params ...string) string {
			if i18nType != Web {
				logger.Errorf("Invalid type for I18n: %s", i18nType)
				return ""
			}
			return localize(localizer, key, params...)
		})
		c.Next()
	}
}

// loadTranslationsFromDisk attempts to load translation files from "web/translation" using the local filesystem.
func loadTranslationsFromDisk(bundle *i18n.Bundle) error {
	root := os.DirFS("web")
	return fs.WalkDir(root, "translation", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(root, path)
		if err != nil {
			return err
		}
		_, err = bundle.ParseMessageFileBytes(data, path)
		return err
	})
}

// parseTranslationFiles parses embedded translation files and adds them to the i18n bundle.
func parseTranslationFiles(i18nFS embed.FS, i18nBundle *i18n.Bundle) error {
	err := fs.WalkDir(i18nFS, "translation",
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			data, err := i18nFS.ReadFile(path)
			if err != nil {
				return err
			}

			_, err = i18nBundle.ParseMessageFileBytes(data, path)
			return err
		})
	if err != nil {
		return err
	}

	return nil
}
