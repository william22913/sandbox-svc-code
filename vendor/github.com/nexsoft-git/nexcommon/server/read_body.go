package server

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// ReadBody reads a request body with NO size limit at all — the
// caller decides how much memory this process allocates. Kept
// unchanged for backward compatibility (it's an exported function;
// an external caller may already depend on this exact signature).
// Prefer ReadBodyWithLimit for any new call site.
func ReadBody(request *http.Request) (output string, bodySize int, err error) {
	byteBody, err := ioutil.ReadAll(request.Body)
	defer request.Body.Close()
	if err != nil {
		return "", 0, errors.New("BODY_INVALID")
	}
	return string(byteBody), len(byteBody), nil
}

// ErrBodyTooLarge is returned by ReadBodyWithLimit specifically when
// the body exceeded maxBytes — distinct from the generic BODY_INVALID
// a malformed/unreadable body reports — so a caller that wants to
// answer a genuinely oversized request differently (e.g. with
// error.ErrRequestBodyTooLarge's 413, instead of a generic 400) has
// something to check for.
var ErrBodyTooLarge = errors.New("BODY_TOO_LARGE")

// ReadBodyWithLimit is ReadBody, capped at maxBytes via
// http.MaxBytesReader. A maxBytes <= 0 means no limit (identical to
// ReadBody).
//
// w is accepted (and may be nil) because http.MaxBytesReader uses it,
// when it implements an internal http.response-only interface, to
// hint the server to close the connection after a too-large body —
// passing the real http.ResponseWriter for the request lets that hint
// actually work; nil degrades to "just enforce the limit," no panic.
func ReadBodyWithLimit(w http.ResponseWriter, request *http.Request, maxBytes int64) (output string, bodySize int, err error) {
	if maxBytes <= 0 {
		return ReadBody(request)
	}

	body := request.Body
	limited := http.MaxBytesReader(w, body, maxBytes)
	request.Body = limited
	defer func() { request.Body = body }()

	byteBody, readErr := ioutil.ReadAll(limited)
	defer limited.Close()
	if readErr != nil {
		// Go's http.MaxBytesReader has used this exact message since
		// long before it gained a distinct error TYPE (*http.MaxBytesError,
		// Go 1.19+) — matching on it rather than that type keeps this
		// working under the older toolchains go.mod's "go 1.16" line
		// still allows building with.
		if strings.Contains(readErr.Error(), "request body too large") {
			return "", 0, ErrBodyTooLarge
		}
		return "", 0, errors.New("BODY_INVALID")
	}
	return string(byteBody), len(byteBody), nil
}

func ReadHeader(request *http.Request, headerName string) (result string) {
	result = request.Header.Get(headerName)
	if result == "" {
		result = request.Header.Get(strings.ToLower(headerName))
	}
	return result
}
