package dmgo

import (
	"encoding/base64"
	"fmt"
)

// coping with golang's json idiosyncracies

// returns value, parent, error
func followJSON(msg map[string]interface{}, pathStart string, path ...string) (interface{}, map[string]interface{}, error) {
	cursor := msg
	var ok bool
	var untyped interface{}
	fullPath := append([]string{pathStart}, path...)
	for i := range fullPath {
		untyped, ok = cursor[fullPath[i]]
		if !ok {
			return nil, nil, fmt.Errorf("followJSON: could not find %v in json path", fullPath[i])
		}
		if i < len(fullPath)-1 {
			cursor, ok = untyped.(map[string]interface{})
			if !ok {
				return nil, nil, fmt.Errorf("followJSON: typing err in json path at %v", fullPath[i])
			}
		}
	}
	return untyped, cursor, nil
}

func getByteSliceFromJSON(untypedVal interface{}) ([]byte, error) {
	base64Str, ok := untypedVal.(string)
	if !ok {
		return nil, fmt.Errorf("getByteSliceFromJSON: json val is not a string")
	}
	bytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("getByteSliceFromJSON: base64 decode error: %v", err)
	}
	return bytes, nil
}
