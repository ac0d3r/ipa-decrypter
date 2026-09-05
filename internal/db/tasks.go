package db

import "strings"

func (d *DB) CreateTask(t Task) error {
	normalizeTaskLogs(&t)
	return d.gorm.Create(&t).Error
}

func (d *DB) UpdateTask(t Task) error {
	normalizeTaskLogs(&t)
	return d.gorm.Save(&t).Error
}

func (d *DB) DeleteTask(id string) error {
	return d.gorm.Delete(&Task{}, "id = ?", id).Error
}

func (d *DB) GetTask(id string) (Task, error) {
	var t Task
	err := d.gorm.First(&t, "id = ?", id).Error
	normalizeTaskLogs(&t)
	return t, err
}

func (d *DB) ListTasks() ([]Task, error) {
	var out []Task
	err := d.gorm.Order("created_at DESC").Find(&out).Error
	for i := range out {
		normalizeTaskLogs(&out[i])
	}
	return out, err
}

func (d *DB) ListActiveTasks() ([]Task, error) {
	var out []Task
	err := d.gorm.Where("status NOT IN ?", []string{"completed", "failed"}).
		Order("created_at ASC").
		Find(&out).Error
	for i := range out {
		normalizeTaskLogs(&out[i])
	}
	return out, err
}

func (d *DB) ListCompletedTasks(limit int) ([]Task, error) {
	if limit <= 0 {
		limit = 20
	}
	var out []Task
	err := d.gorm.Where("status = ?", "completed").
		Order("updated_at DESC").
		Limit(limit).
		Find(&out).Error
	for i := range out {
		normalizeTaskLogs(&out[i])
	}
	return out, err
}

func (d *DB) ListCompletedTasksForBundles(region string, bundleIDs []string) ([]Task, error) {
	if len(bundleIDs) == 0 {
		return nil, nil
	}
	var out []Task
	q := d.gorm.Where("status = ? AND bundle_id IN ? AND output_path != ''", "completed", bundleIDs)
	if region != "" {
		q = q.Where("region = ?", strings.ToLower(region))
	}
	err := q.Order("updated_at DESC").Find(&out).Error
	for i := range out {
		normalizeTaskLogs(&out[i])
	}
	return out, err
}

func (d *DB) ListCompletedTasksForVersion(bundleID, region, externalVersionID, displayVersion string) ([]Task, error) {
	if externalVersionID == "" && displayVersion == "" {
		return nil, nil
	}
	var out []Task
	q := d.gorm.Where(
		"status = ? AND bundle_id = ? AND region = ? AND output_path != ''",
		"completed", bundleID, strings.ToLower(region),
	)
	if externalVersionID != "" {
		q = q.Where("external_version_id = ?", externalVersionID)
	} else {
		q = q.Where("display_version = ?", displayVersion)
	}
	err := q.Order("updated_at DESC").Find(&out).Error
	for i := range out {
		normalizeTaskLogs(&out[i])
	}
	return out, err
}

func (d *DB) ListFailedTasksForVersion(bundleID, region, externalVersionID string) ([]Task, error) {
	var out []Task
	err := d.gorm.Where(
		"status = ? AND bundle_id = ? AND region = ? AND external_version_id = ? AND encrypted_path != ''",
		"failed", bundleID, strings.ToLower(region), externalVersionID,
	).Order("updated_at DESC").Find(&out).Error
	for i := range out {
		normalizeTaskLogs(&out[i])
	}
	return out, err
}
