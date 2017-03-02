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

func getByteSliceFromJSON(msg map[string]interface{}, pathStart string, path ...string) ([]byte, error) {
	untypedVal, _, err := followJSON(msg, pathStart, path...)
	if err != nil {
		return nil, fmt.Errorf("getByteSlice: %v", err)
	}
	base64Str, ok := untypedVal.(string)
	if !ok {
		return nil, fmt.Errorf("getByteSlice: string not found at end of path")
	}
	bytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("getByteSlice: base64 decode error: %v", err)
	}
	return bytes, nil
}

func replaceNodeInJSON(msg map[string]interface{}, payload interface{}, pathStart string, path ...string) error {
	_, parent, err := followJSON(msg, pathStart, path...)
	if err != nil {
		return fmt.Errorf("replaceNode: %v", err)
	}
	fullPath := append([]string{pathStart}, path...)
	lastFieldName := fullPath[len(fullPath)-1]
	parent[lastFieldName] = payload
	return nil
}

func replaceByteSliceInJSON(msg map[string]interface{}, payload []byte, pathStart string, path ...string) error {
	return replaceNodeInJSON(msg, base64.StdEncoding.EncodeToString(payload), pathStart, path...)
}
