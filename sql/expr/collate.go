package expr

import "github.com/RevenueMonster/sqlike/sqlike/primitive"

// Collate :
func Collate(collate string, col interface{}, charset ...string) (o primitive.Encoding) {
	if len(charset) > 0 {
		o.Charset = &charset[0]
	}
	o.Column = col
	o.Collate = collate
	return
}
