package client

import "encoding/json"

// parseJSON is a helper to unmarshal JSON data.
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
