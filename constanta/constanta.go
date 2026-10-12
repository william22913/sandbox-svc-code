package constanta

// Field-label keys. Each one must have an entry in every shipped bundle under
// i18n/common/constanta/ — a missing entry renders the raw key to the client.
const (
	ItemID   = "ITEM_ID"
	Favorite = "FAVORITE"
)

// FieldLabelKeys is every field label this service defines. The bundle guard
// test (tests/bundle_coverage_test.go) walks this.
var FieldLabelKeys = []string{ItemID, Favorite}

// RuleNameKeys are the rule-name strings a DTO's min/max/enum tags can produce.
// nexcommon's validator passes them with IsConverted=true, so they resolve
// through common.constanta — and this service ships its own bundles in place of
// nexcommon's, so each one has to be present here or the client sees the raw
// rule name.
var RuleNameKeys = []string{
	"NEED_MORE_THAN",
	"NEED_LESS_THAN",
	"EQUAL",
	"FIXED_VALUE",
}

// Swagger tag for this feature's endpoints, so the generated document groups
// them by owning controller rather than landing in one undifferentiated pile.
const TagFavorite = "Favorite"
