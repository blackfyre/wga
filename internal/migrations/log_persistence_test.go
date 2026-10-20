package migrations

import "testing"

func TestLogPersistenceCapsRetentionAtAWeek(t *testing.T) {
	cases := []struct {
		name    string
		current int
		want    int
	}{
		{name: "longer retention is capped", current: 30, want: 7},
		{name: "shorter retention is kept", current: 3, want: 3},
		{name: "disabled persistence is kept", current: 0, want: 0},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			app := newMigrationTestApp(t, t.TempDir())
			t.Cleanup(func() {
				if err := app.ClearBootstrap(); err != nil {
					t.Error(err)
				}
			})
			app.Settings().Logs.MaxDays = test.current
			if err := app.Save(app.Settings()); err != nil {
				t.Fatalf("save settings: %v", err)
			}

			if err := boundLogPersistence(app); err != nil {
				t.Fatalf("bound log persistence: %v", err)
			}
			if err := app.ReloadSettings(); err != nil {
				t.Fatalf("reload settings: %v", err)
			}

			if got := app.Settings().Logs.MaxDays; got != test.want {
				t.Errorf("logs max days = %d; want %d", got, test.want)
			}
		})
	}
}
