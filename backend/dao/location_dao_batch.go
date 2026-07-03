package dao

import "time"

// LatestLocationAge — возраст последней GPS-точки пользователя.
type LatestLocationAge struct {
	UserID     int
	AgeSeconds int64
}

// GetLatestAgePerUser возвращает age_seconds последней точки для каждого user_id (один SQL).
func (dao *LocationDAO) GetLatestAgePerUser() ([]LatestLocationAge, error) {
	var rows []LatestLocationAge
	err := dao.DB.Raw(`
		SELECT DISTINCT ON (user_id) user_id,
			GREATEST(0, EXTRACT(EPOCH FROM (NOW() - COALESCE(captured_at, created_at))))::bigint AS age_seconds
		FROM locations
		ORDER BY user_id, COALESCE(captured_at, created_at) DESC
	`).Scan(&rows).Error
	return rows, err
}
