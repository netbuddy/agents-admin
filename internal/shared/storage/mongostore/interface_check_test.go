package mongostore

import (
	"agents-admin/internal/shared/storage"
)

// Compile-time interface check
var _ storage.PersistentStore = (*Store)(nil)
