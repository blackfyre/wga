package migrations

import (
	"errors"
	"sync"

	"github.com/blackfyre/wga/internal/config"
)

var (
	migrationConfigMu  sync.RWMutex
	migrationConfig    config.Migrations
	migrationConfigSet bool
)

func Configure(values config.Migrations) error {
	migrationConfigMu.Lock()
	defer migrationConfigMu.Unlock()

	if migrationConfigSet {
		return errors.New("migration configuration already initialised")
	}

	migrationConfig = values
	migrationConfigSet = true

	return nil
}

func configuredMigrations() (config.Migrations, error) {
	migrationConfigMu.RLock()
	defer migrationConfigMu.RUnlock()

	if !migrationConfigSet {
		return config.Migrations{}, errors.New("migration configuration is not initialised")
	}

	return migrationConfig, nil
}
