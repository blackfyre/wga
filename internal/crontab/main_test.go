package crontab

import (
	"context"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/cron"
)

type noopRefreshJob struct{}

func (noopRefreshJob) Run(context.Context, string) error { return nil }

func TestRegisterCronJobsRegistersItineraryPurge(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset app: %v", err)
		}
	})

	values := map[string]string{
		"WGA_ENV":            "development",
		"WGA_PROTOCOL":       "http",
		"WGA_HOSTNAME":       "localhost:8090",
		"WGA_SENDER_NAME":    "WGA",
		"WGA_SENDER_ADDRESS": "sender@example.test",
	}
	cfg := config.LoadFrom(func(key string) string { return values[key] })

	server, err := cfg.Server()
	if err != nil {
		t.Fatalf("load server config: %v", err)
	}
	sitemap, err := cfg.Sitemap()
	if err != nil {
		t.Fatalf("load sitemap config: %v", err)
	}

	RegisterCronJobs(app, server.Postcards, sitemap, noopRefreshJob{})

	jobs := app.Cron().Jobs()
	ids := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		ids[job.Id()] = true
	}

	for _, want := range []string{"postcards", "contributors-refresh", "sitemap", "itineraries-purge", "participation-purge"} {
		if !ids[want] {
			t.Errorf("expected cron job %q to be registered; registered jobs: %v", want, jobIDs(jobs))
		}
	}
}

func jobIDs(jobs []*cron.Job) []string {
	ids := make([]string, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.Id())
	}
	return ids
}
