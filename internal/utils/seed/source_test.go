package seed

import "testing"

func TestSafeRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "nested path", value: "music/track.mp3", want: "music/track.mp3"},
		{name: "normalised nested path", value: "music/../music/track.mp3", want: "music/track.mp3"},
		{name: "empty path", value: "", wantErr: true},
		{name: "absolute path", value: "/music/track.mp3", wantErr: true},
		{name: "parent directory", value: "..", wantErr: true},
		{name: "parent traversal", value: "../music/track.mp3", wantErr: true},
		{name: "normalised traversal", value: "music/../../track.mp3", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := safeRelativePath(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("safeRelativePath(%q): %v", test.value, err)
			}
			if got != test.want {
				t.Fatalf("safeRelativePath(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
