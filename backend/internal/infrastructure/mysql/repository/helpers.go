package repository

import "encoding/json"

func marshalJSON(value any) ([]byte, error) {
	if value == nil {
		return []byte("null"), nil
	}

	return json.Marshal(value)
}

func unmarshalStringSlice(raw []byte) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}
	}

	var result []string
	if err := json.Unmarshal(raw, &result); err != nil {
		return []string{}
	}

	return result
}

func unmarshalStringMap(raw []byte) map[string]string {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]string{}
	}

	var result map[string]string
	if err := json.Unmarshal(raw, &result); err != nil {
		return map[string]string{}
	}

	return result
}
