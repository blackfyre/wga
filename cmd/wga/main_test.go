package main

import "testing"

func TestCommandCapabilityFor(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want commandCapability
	}{
		{name: "default server", want: commandNeedsServer},
		{name: "server flags", args: []string{"--dev"}, want: commandNeedsServer},
		{name: "serve", args: []string{"serve"}, want: commandNeedsServer},
		{name: "sitemap", args: []string{"generate-sitemap"}, want: commandNeedsSitemap},
		{name: "migration", args: []string{"migrate", "up"}, want: commandNeedsNothing},
		{name: "migration collections", args: []string{"migrate", "collections"}, want: commandNeedsNothing},
		{name: "music URLs", args: []string{"generate-music-urls"}, want: commandNeedsNothing},
		{name: "unknown command", args: []string{"not-a-command"}, want: commandNeedsNothing},
		{name: "server data directory", args: []string{"--dir", "test_data"}, want: commandNeedsServer},
		{name: "migration data directory", args: []string{"--dir", "test_data", "migrate", "up"}, want: commandNeedsNothing},
		{name: "help", args: []string{"serve", "--help"}, want: commandNeedsNothing},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := commandCapabilityFor(test.args); got != test.want {
				t.Fatalf("expected capability %d, got %d", test.want, got)
			}
		})
	}
}
