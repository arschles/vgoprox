package rdbms

import (
	"github.com/gomods/athens/pkg/storage/rdbms/models"
)

// Exists checks for a specific version of a module
func (r *ModuleStore) Exists(module, vsn string) bool {
	result := models.Module{}
	query := r.conn.Where("module = ?", module).Where("version = ?", vsn)
	count, err := query.Count(&result)

	return err == nil && count > 0
}
