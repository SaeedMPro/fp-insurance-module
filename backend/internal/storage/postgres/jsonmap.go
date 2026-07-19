package postgres

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// jsonMap adapts an arbitrary JSON object to a Postgres jsonb column.
type jsonMap map[string]any

func (m jsonMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *jsonMap) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		s, sok := value.(string)
		if !sok {
			return errors.New("jsonMap: unsupported scan type")
		}
		bytes = []byte(s)
	}
	if len(bytes) == 0 {
		*m = nil
		return nil
	}
	return json.Unmarshal(bytes, m)
}
