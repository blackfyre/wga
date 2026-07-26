package contributors

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase/core"
)

const snapshotKey = "github"

//go:embed fallback.json
var fallbackContributors []byte

type Store struct {
	app      core.App
	fallback []Contributor
}

func NewStore(app core.App) (*Store, error) {
	var fallback []Contributor
	if err := json.Unmarshal(fallbackContributors, &fallback); err != nil {
		return nil, fmt.Errorf("decode contributor fallback: %w", err)
	}
	if len(fallback) == 0 {
		return nil, errors.New("contributor fallback is empty")
	}

	return &Store{app: app, fallback: fallback}, nil
}

func (s *Store) Current(_ context.Context) (Snapshot, error) {
	record, err := s.app.FindFirstRecordByData(constants.CollectionContributorSnapshots, "key", snapshotKey)
	if errors.Is(err, sql.ErrNoRows) {
		return Snapshot{Contributors: s.fallback, Source: SnapshotSourceFileFallback}, nil
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("find contributor snapshot: %w", err)
	}

	contributors, err := decodeContributors(record.GetRaw("payload"))
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{Contributors: contributors, Source: SnapshotSourceCache}, nil
}

func (s *Store) Replace(app core.App, contributors []Contributor) error {
	if len(contributors) == 0 {
		return errors.New("cannot store empty contributor snapshot")
	}

	record, err := app.FindFirstRecordByData(constants.CollectionContributorSnapshots, "key", snapshotKey)
	if errors.Is(err, sql.ErrNoRows) {
		collection, collectionErr := app.FindCollectionByNameOrId(constants.CollectionContributorSnapshots)
		if collectionErr != nil {
			return collectionErr
		}
		record = core.NewRecord(collection)
		record.Set("key", snapshotKey)
	} else if err != nil {
		return err
	}

	record.Set("payload", contributors)
	return app.Save(record)
}

func decodeContributors(raw any) ([]Contributor, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode contributor snapshot: %w", err)
	}

	var contributors []Contributor
	if err := json.Unmarshal(encoded, &contributors); err != nil {
		return nil, fmt.Errorf("decode contributor snapshot: %w", err)
	}
	if len(contributors) == 0 {
		return nil, errors.New("contributor snapshot is empty")
	}

	return contributors, nil
}
