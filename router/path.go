package router

// V1PrefixPath is the shared route prefix every {entity}_endpoint registers
// under, so the version of the API lives in one place rather than in each
// endpoint package's string literals.
const V1PrefixPath = "/v1"
