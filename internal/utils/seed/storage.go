package seed

import (
	"errors"
	"fmt"
	iofs "io/fs"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func SeedStorage(app core.App, options SourceOptions) error {
	paths, err := resolveSourcePaths(options)
	if err != nil {
		return err
	}
	defer func() {
		_ = paths.Close()
	}()

	data, err := loadSourceData(paths)
	if err != nil {
		return err
	}
	if err := loadSourceFiles(paths.storage, &data); err != nil {
		return err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return err
	}
	defer func() {
		_ = fsys.Close()
	}()

	if err := seedArtworkStorage(app, fsys, paths, data); err != nil {
		return err
	}

	return seedMusicStorage(app, fsys, paths, data)
}

func seedArtworkStorage(app core.App, fsys *filesystem.System, paths sourcePaths, data sourceData) error {
	for _, artwork := range data.artworks {
		filename := data.artworkFiles[artwork.ID]
		record, err := app.FindRecordById(constants.CollectionArtworks, artwork.ID)
		if err != nil {
			return fmt.Errorf("find artwork %q: %w", artwork.ID, err)
		}
		if err := uploadSourceFile(paths.storage, fsys, sourceFilePath("Artworks", artwork.ID, filename), record.BaseFilesPath()+"/"+filename); err != nil {
			return fmt.Errorf("upload artwork %q: %w", artwork.ID, err)
		}
		record.Set("image", filename)
		if err := app.SaveNoValidate(record); err != nil {
			return fmt.Errorf("save artwork %q image: %w", artwork.ID, err)
		}
	}

	return nil
}

func seedMusicStorage(app core.App, fsys *filesystem.System, paths sourcePaths, data sourceData) error {
	for _, track := range data.musicTracks {
		filename := data.musicFiles[track.ID]
		record, err := app.FindRecordById("music_song", track.ID)
		if err != nil {
			return fmt.Errorf("find music track %q: %w", track.ID, err)
		}
		sourcePath, err := sourceMusicFilePath(track)
		if err != nil {
			return fmt.Errorf("resolve music track %q source: %w", track.ID, err)
		}
		if err := uploadSourceFile(paths.storage, fsys, sourcePath, record.BaseFilesPath()+"/"+filename); err != nil {
			return fmt.Errorf("upload music track %q: %w", track.ID, err)
		}
		record.Set("source", filename)
		if err := app.SaveNoValidate(record); err != nil {
			return fmt.Errorf("save music track %q source: %w", track.ID, err)
		}
	}

	return nil
}

func uploadSourceFile(source iofs.FS, fsys *filesystem.System, sourcePath string, targetPath string) error {
	content, err := iofs.ReadFile(source, sourcePath)
	if err != nil {
		return err
	}
	if len(content) == 0 {
		return errors.New("source file is empty")
	}

	return fsys.Upload(content, targetPath)
}
