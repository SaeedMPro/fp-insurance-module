package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONMap adapts an arbitrary JSON object to a Postgres jsonb column.
type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		if s, ok := value.(string); ok {
			bytes = []byte(s)
		} else {
			return errors.New("JSONMap: unsupported scan type")
		}
	}
	if len(bytes) == 0 {
		*m = nil
		return nil
	}
	return json.Unmarshal(bytes, m)
}
