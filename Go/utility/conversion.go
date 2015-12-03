// Conversion utilities
package utility

import (
	"fmt"
	"unsafe"

	"github.com/ugorji/go/codec"
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

// IfcToJSON encodes interface{} to a JSON byte slice
func IfcToJSON(ifcVal interface{}) (jsonOut []byte, err error) {
	jsonOut = make([]byte, 0, unsafe.Sizeof(ifcVal))
	handle := new(codec.JsonHandle)
	encoder := codec.NewEncoderBytes(&(jsonOut), handle)
	err = encoder.Encode(ifcVal)
	return
}

// JSONToIfc decodes JSON byte array to map[string]interface{}
func JSONToIfc(jsonIn []byte) (ifcVal map[string]interface{}, err error) {
	handle := new(codec.JsonHandle)
	decoder := codec.NewDecoderBytes(jsonIn, handle)
	err = decoder.Decode(&ifcVal)
	return
}