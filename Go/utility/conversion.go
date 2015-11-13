// Conversion utilities
package utility

import (
	"fmt"
)

// ConvertToMsgCode converts interface{} values with the types that typically underly JSON-encoded integers
func ConvertToString(ifcVal interface{}) (string) {
	switch val := ifcVal.(type) {
		case string:
			return val
		case []uint8:
			return string(val)
		default:
			return "UNKNOWN MESSAGE TYPE"
	}
}

// ConvertToMsgCode converts interface{} values with the types that typically underly JSON-encoded integers
func TryConvertToString(ifcVal interface{}) (strVal string, e error) {
	e = nil
	switch val := ifcVal.(type) {
		case string:
			strVal = val
			return
		case []uint8:
			strVal = string(val)
			return
		default:
			strVal = ""
			e = fmt.Errorf("Value cannot be converted to a string")
			return
	}
}