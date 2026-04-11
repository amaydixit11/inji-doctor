// Package checker provides shared helper functions used across all checkers.
package checker

import "encoding/json"

// unmarshalJSON is a shared JSON unmarshal helper.
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
