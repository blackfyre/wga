package migrations

import "testing"

func TestLogPersistenceRetainsLogsForAWeek(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ClearBootstrap(); err != nil {
			t.Error(err)
		}
	})
	app.Settings().Logs.MaxDays = 30

	if err := boundLogPersistence(app); err != nil {
		t.Fatalf("bound log persistence: %v", err)
	}
	if err := app.ReloadSettings(); err != nil {
		t.Fatalf("reload settings: %v", err)
	}

	if got := app.Settings().Logs.MaxDays; got != 7 {
		t.Errorf("logs max days = %d; want 7", got)
	}
}
