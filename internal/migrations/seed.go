package migrations

import (
	"errors"
	"io/fs"

	"github.com/blackfyre/wga/internal/assets"
)

var seedFiles fs.FS = assets.InternalFiles

func readSeedFile(name string) ([]byte, bool, error) {
	data, err := fs.ReadFile(seedFiles, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	return data, true, nil
}
