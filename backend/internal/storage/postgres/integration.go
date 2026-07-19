package postgres

import "context"

// ActiveAPIKeyExists reports whether an active integration API key matches the
// given SHA-256 hash.
func (s *Store) ActiveAPIKeyExists(ctx context.Context, keyHash string) (bool, error) {
	var count int64
	err := s.ctx(ctx).Model(&apiKeyRow{}).
		Where("api_key_hash = ? AND is_active = true", keyHash).
		Count(&count).Error
	return count > 0, err
}
