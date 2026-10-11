package bundles

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/nexsoft-git/nexlogger/log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

func NewBundles(
	rootDir string,
	defaultLanguage string,
) (
	Bundles,
	error,
) {
	bundles := &bundles{
		defaultLanguage: defaultLanguage,
	}
	err := bundles.loadBundleI18N(rootDir)
	return bundles, err
}

type bundles struct {
	bundles         map[string]*i18n.Bundle
	defaultLanguage string
}

/*
[PARAM]

	@bundleName : nama bundle yang ada di struktur i18n
	bundleName yang dapat digunakan pada contoh dibawah adalah ["common.constanta", "common.error", "common_service"]
	|- common
		|-- constanta
		|-- error
	|- common_service

	@messageID : key yang terdapat pada file i18n
	@language : bahasa yang akan digunakan untuk memperoleh message
	@param : param yang akan diisikan untuk melengkapi message yang digenerate
*/
func (b bundles) ReadMessageBundle(
	bundleName string,
	messageID string,
	language string,
	param map[string]interface{},
) (output string) {

	defer func() {
		if r := recover(); r != nil {
			output = messageID
		}
	}()

	if language == "" {
		language = b.defaultLanguage
	}

	bundle := b.bundles[bundleName]
	localize := i18n.NewLocalizer(bundle, language)

	if param == nil {
		output = localize.MustLocalize(&i18n.LocalizeConfig{
			MessageID: messageID,
		})
	} else {
		output = localize.MustLocalize(&i18n.LocalizeConfig{
			MessageID:    messageID,
			TemplateData: param,
		})
	}

	return
}

// HasMessage — see the Bundles interface doc comment. Uses Localize
// (returns an error) rather than MustLocalize (panics), so this can
// check existence without needing a recover() and without the
// ambiguity of "did ReadMessageBundle return the real translation, or
// did it fall back to echoing the raw key because nothing matched."
//
// Deliberately checks for *i18n.MessageNotFoundErr specifically,
// rather than treating any non-nil error as "missing": a real bundle
// entry's template (e.g. "Field {{.FieldName}} Cannot Empty") fails
// to EXECUTE with no TemplateData supplied, which Localize also
// surfaces as a non-nil error even though the key itself is present —
// an existence check has to tell those two failure modes apart, or
// every templated entry in the whole bundle would wrongly report as
// missing.
func (b bundles) HasMessage(
	bundleName string,
	messageID string,
	language string,
) bool {
	if language == "" {
		language = b.defaultLanguage
	}

	bundle, ok := b.bundles[bundleName]
	if !ok {
		return false
	}

	localize := i18n.NewLocalizer(bundle, language)
	_, err := localize.Localize(&i18n.LocalizeConfig{MessageID: messageID})
	if err == nil {
		return true
	}

	var notFound *i18n.MessageNotFoundErr
	return !stderrors.As(err, &notFound)
}

// ValidateKeysExist — see the Bundles interface doc comment.
func (b bundles) ValidateKeysExist(
	bundleName string,
	keys []string,
	language string,
) (missing []string) {
	for _, key := range keys {
		if !b.HasMessage(bundleName, key, language) {
			missing = append(missing, key)
		}
	}
	return
}

func (b *bundles) loadBundleI18N(dirName string) error {
	b.bundles = make(map[string]*i18n.Bundle)
	root := fmt.Sprintf("./%s/", dirName)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.Name() == dirName {
			return nil
		}

		if info.IsDir() {
			path = strings.ReplaceAll(path, "\\", "/")
			pathSplit := strings.Split(path, "/")
			name := strings.Join(pathSplit[1:], ".")
			filePath := fmt.Sprintf("./%s", path)

			dir, err := ioutil.ReadDir(filePath)
			if err != nil {
				log.Fatal().
					Err(err).
					Caller().
					Msg("Error When reading i18n bundle.")
			}

			bundle := i18n.NewBundle(language.Indonesian)
			bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

			found := false
			for _, f := range dir {
				if !f.IsDir() {
					found = true

					i18nFilePath := fmt.Sprintf("%s/%s", filePath, f.Name())
					_, err := bundle.LoadMessageFile(i18nFilePath)

					if err != nil {
						return err
					}
				}
			}

			if found {
				b.bundles[name] = bundle
			}
		}
		return err
	})

	return err
}
