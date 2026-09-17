package repositories

import (
	"context"

	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const dualModeReferenceCacheKey = "dual-mode:reference"

// DualModeReference is the bounded persisted-data projection shared by Dual
// Mode requests.
type DualModeReference struct {
	Schools []ArtistSchoolOption
	Periods []ArtistPeriodOption
	BornMin int
	BornMax int
}

// DualModeReferenceRepository loads the application-scoped Dual Mode reference
// projection while preserving caller cancellation.
type DualModeReferenceRepository struct {
	app  core.App
	ctx  context.Context
	load func(context.Context) (DualModeReference, error)
}

// NewDualModeReferenceRepository creates a context-free compatibility reader.
func NewDualModeReferenceRepository(app core.App) *DualModeReferenceRepository {
	return NewDualModeReferenceRepositoryWithContext(context.Background(), app)
}

// NewDualModeReferenceRepositoryWithContext links cold loads and shared waits to
// the request context.
func NewDualModeReferenceRepositoryWithContext(ctx context.Context, app core.App) *DualModeReferenceRepository {
	return &DualModeReferenceRepository{
		app: app,
		ctx: ctx,
		load: func(ctx context.Context) (DualModeReference, error) {
			return loadDualModeReference(ctx, app)
		},
	}
}

// Load returns a defensive copy of the current reference projection.
func (r *DualModeReferenceRepository) Load() (DualModeReference, error) {
	value, err := utils.GetOrLoadInstrumentedCachedValue(
		r.ctx,
		r.app,
		dualModeReferenceCacheKey,
		0,
		utils.CacheDualModeReference,
		func() (DualModeReference, error) { return r.load(r.ctx) },
	)
	if err != nil {
		return DualModeReference{}, err
	}
	return cloneDualModeReference(value), nil
}

// InvalidateDualModeReference advances the projection generation so the next
// request observes current persisted data.
func InvalidateDualModeReference(app core.App) {
	utils.DeleteInstrumentedCachedValue(context.Background(), app, dualModeReferenceCacheKey, utils.CacheDualModeReference)
}

func loadDualModeReference(ctx context.Context, app core.App) (DualModeReference, error) {
	if err := context.Cause(ctx); err != nil {
		return DualModeReference{}, err
	}
	schools, err := ListArtistSchools(app)
	if err != nil {
		return DualModeReference{}, err
	}
	if err := context.Cause(ctx); err != nil {
		return DualModeReference{}, err
	}
	periods, err := ListArtistPeriods(app)
	if err != nil {
		return DualModeReference{}, err
	}
	if err := context.Cause(ctx); err != nil {
		return DualModeReference{}, err
	}
	bornMin, bornMax, err := NewArtistIndexRepositoryWithContext(ctx, app).BirthYearBounds()
	if err != nil {
		return DualModeReference{}, err
	}
	return DualModeReference{Schools: schools, Periods: periods, BornMin: bornMin, BornMax: bornMax}, nil
}

func cloneDualModeReference(value DualModeReference) DualModeReference {
	value.Schools = append([]ArtistSchoolOption(nil), value.Schools...)
	value.Periods = append([]ArtistPeriodOption(nil), value.Periods...)
	return value
}
