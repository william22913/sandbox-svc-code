package bundles

type Bundles interface {
	ReadMessageBundle(
		bundleName string,
		messageID string,
		language string,
		param map[string]interface{},
	) (output string)

	// HasMessage reports whether messageID actually has an entry in
	// bundleName for language (empty language uses the bundle's own
	// default), without the fallback-to-raw-key behavior
	// ReadMessageBundle has to have for its normal callers. Use this
	// (or ValidateKeysExist) at startup to catch a missing bundle
	// entry as a loud failure instead of a client silently seeing the
	// raw key/code in a response.
	HasMessage(
		bundleName string,
		messageID string,
		language string,
	) bool

	// ValidateKeysExist checks every key in keys against bundleName
	// for language and returns the ones that have no entry — empty
	// means every key is covered. Built for a fail-fast startup check:
	// a service registers its validator rule names and reads
	// error.AllCodes(), then calls this once per bundle/language
	// combination it ships and refuses to start if anything comes
	// back non-empty, instead of finding out from a client's response
	// containing a raw code with no translation.
	ValidateKeysExist(
		bundleName string,
		keys []string,
		language string,
	) (missing []string)
}
