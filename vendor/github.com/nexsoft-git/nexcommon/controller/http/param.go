package http

import (
	"net/http"
	"strings"

	"github.com/nexsoft-git/nexcommon/docs/swagger"
)

func NewHandleFuncParam(
	path string,
	ws WrapperServiceResult,
	method ...string,
) *handleFuncParam {
	// Only allocate a fresh method map when this path hasn't been
	// registered yet. This used to run unconditionally on every call,
	// so a path with two methods (GET /teachers, then POST /teachers)
	// had the SECOND registration wipe the first's method map before
	// adding its own — the earlier method silently vanished from the
	// generated swagger document, with no error anywhere.
	if swagger.SwaggerContainer.SwaggerPath[path] == nil {
		swagger.SwaggerContainer.SwaggerPath[path] = make(swagger.SwaggerControllerMethods)
	}

	if strings.Contains(strings.ToLower(strings.Join(method, ", ")), "get") {
		ws.cs.RequestBody = nil
	}

	for i := 0; i < len(method); i++ {
		if method[i] != http.MethodOptions {
			swagger.SwaggerContainer.SwaggerPath[path][strings.ToLower(method[i])] = ws.cs
		}
	}

	return &handleFuncParam{
		path:   path,
		f:      ws.f,
		method: method,
	}
}

// resolvePathParamSwaggerType decides a path param's generated swagger
// "type" and its "min" example value. Shared by WrapService and
// WrapServiceListData's near-identical loops (was duplicated inline in
// both before this change — a single source avoids the two silently
// drifting from each other the next time either needs a fix).
//
// If declared has an explicit entry for name (via PathParamTypes), it
// wins. Otherwise this falls back to the original name-based
// heuristic: any param whose name contains "id" (case-insensitive)
// and isn't literally "uuid" is assumed to be an integer database id.
// That heuristic is wrong for every uuid-keyed param not literally
// named "uuid" — e.g. {eduId}, {teacherId} — which is exactly why
// PathParamTypes exists: to let a caller correct it per param without
// this helper needing to guess.
func resolvePathParamSwaggerType(name string, declared map[string]string) (dataType string, min int) {
	if t, ok := declared[name]; ok {
		if t == "integer" {
			return t, 1
		}
		return t, 0
	}
	if name != "uuid" && strings.Contains(strings.ToLower(name), "id") {
		return "integer", 1
	}
	return "string", 0
}

type handleFuncParam struct {
	path   string
	f      func(http.ResponseWriter, *http.Request)
	method []string
}