package error

import "sync"

var (
	registeredCodesMu sync.Mutex
	registeredCodes   []string
)

func NewUnBundledErrorMessages(
	status int,
	code error,
	f Converter,
) *UnbundledErrorMessages {
	registeredCodesMu.Lock()
	registeredCodes = append(registeredCodes, code.Error())
	registeredCodesMu.Unlock()

	return &UnbundledErrorMessages{
		status:   status,
		code:     code,
		function: f,
		param: []interface{}{
			"UNDEFINED",
			"UNDEFINED",
			"UNDEFINED",
			"UNDEFINED",
			"UNDEFINED",
		},
	}
}

// AllCodes returns every error code ever registered through
// NewUnBundledErrorMessages, in registration order — this package's
// own codes (declared in message.go as package-level vars, which Go
// initializes before any other code runs) AND any project-local
// codes a consumer declares the same way in their own error.go, since
// both go through this same constructor.
//
// This exists specifically so a bundle-coverage guard test can read
// the REAL, complete list of codes a process can emit, instead of a
// hand-maintained mirror of it. A real, shipped bug traced back to
// exactly that gap: a consumer's own guard test walked a hand-typed
// list of "every code nexcommon can emit" that had silently drifted
// from the library — it asserted a code (E-3-SBX-AUT-001) that nexcommon
// does not actually emit, missed the one it does (E-1-CMD-AUT-001),
// and passed anyway, because the list it walked was the same wrong
// list someone had written by hand. Call AllCodes() instead of
// maintaining that list yourself; it cannot drift because it isn't a
// second copy of the truth; it's the truth.
func AllCodes() []string {
	registeredCodesMu.Lock()
	defer registeredCodesMu.Unlock()

	out := make([]string, len(registeredCodes))
	copy(out, registeredCodes)
	return out
}
