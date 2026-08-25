package postcards

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestPostcardsCommandEvaluatesTokenKeyringOnlyForRewrapExecution(t *testing.T) {
	app := testutils.NewTestApp(t)
	installPostcardSchema(t, app)
	calls := 0
	provider := func() (config.PostcardTokenKeyring, error) {
		calls++
		return config.PostcardTokenKeyring{}, errors.New("token keyring unavailable")
	}

	help := newPostcardsCommand(app, provider)
	help.SetOut(io.Discard)
	help.SetErr(io.Discard)
	help.SetArgs([]string{"--help"})
	if err := help.Execute(); err != nil {
		t.Fatalf("postcards help: %v", err)
	}
	if calls != 0 {
		t.Fatalf("help evaluated token keyring %d times", calls)
	}

	inspect := newPostcardsCommand(app, provider)
	inspect.SetOut(io.Discard)
	inspect.SetErr(io.Discard)
	inspect.SetArgs([]string{"inspect"})
	if err := inspect.Execute(); err != nil {
		t.Fatalf("postcards inspect: %v", err)
	}
	if calls != 0 {
		t.Fatalf("inspect evaluated token keyring %d times", calls)
	}

	rewrap := newPostcardsCommand(app, provider)
	rewrap.SetOut(io.Discard)
	rewrap.SetErr(io.Discard)
	rewrap.SetArgs([]string{"rewrap-token-key", "--from", "old"})
	if err := rewrap.Execute(); err == nil {
		t.Fatal("expected rewrap keyring error")
	}
	if calls != 1 {
		t.Fatalf("rewrap evaluated token keyring %d times, want 1", calls)
	}
}

func TestPostcardsRewrapCommandPrintsAggregateCountOnly(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	oldKey := bytes.Repeat([]byte{0x51}, 32)
	newKey := bytes.Repeat([]byte{0x52}, 32)
	oldOnly := testPostcardKeyring(t, "old", map[string][]byte{"old": oldKey})
	rotation := testPostcardKeyring(t, "new", map[string][]byte{"old": oldKey, "new": newKey})
	result, err := QueueWithAccess(app, oldOnly, QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
	}, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}

	command := newPostcardsCommand(app, func() (config.PostcardTokenKeyring, error) { return rotation, nil })
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"rewrap-token-key", "--from", "old", "--limit", "1"})
	if err := command.Execute(); err != nil {
		t.Fatalf("execute rewrap command: %v", err)
	}
	if got, want := output.String(), "rewrapped=1\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
	for _, secret := range []string{result.Access[0].Token, result.Access[0].DeliveryID, "old", "new"} {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("aggregate output exposed sensitive material")
		}
	}
}
