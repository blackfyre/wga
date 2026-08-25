package itineraries

import "testing"

func TestStatusStateMachine(t *testing.T) {
	if got, want := ValidStatus("draft"), true; got != want {
		t.Errorf("ValidStatus(draft) = %v, want %v", got, want)
	}
	if got := ValidStatus("bogus"); got {
		t.Errorf("ValidStatus(bogus) = true, want false")
	}

	// Publication is the only application-driven transition and moves a draft
	// straight to approved so the token is immediately readable.
	if !CanTransition(StatusDraft, StatusApproved) {
		t.Error("draft -> approved must be allowed (immediate publication)")
	}
	if CanTransition(StatusDraft, StatusPending) {
		t.Error("draft -> pending must not be produced by new code")
	}
	if CanTransition(StatusDraft, StatusDraft) {
		t.Error("draft -> draft must not be modelled as a transition")
	}
	// The legacy pending state remains ingress for the backfill and operators.
	if !CanTransition(StatusPending, StatusApproved) {
		t.Error("pending -> approved must be allowed (legacy backfill)")
	}
	if !CanTransition(StatusPending, StatusRejected) {
		t.Error("pending -> rejected must be allowed")
	}
	// Approved records may still be rejected by an operator.
	if !CanTransition(StatusApproved, StatusRejected) {
		t.Error("approved -> rejected must be allowed (moderation)")
	}
	if CanTransition(StatusRejected, StatusApproved) {
		t.Error("rejected is terminal; no further transitions are allowed")
	}

	if !IsPublicStatus("approved") {
		t.Error("approved must be public")
	}
	if IsPublicStatus("pending") || IsPublicStatus("draft") || IsPublicStatus("rejected") {
		t.Error("only approved may be public")
	}
}

func TestSanitiseText(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "A quiet note", want: "A quiet note"},
		{name: "strips script", in: `<script>alert(1)</script>Hello`, want: "Hello"},
		{name: "strips tags", in: `<b>bold</b> text`, want: "bold text"},
		{name: "trims", in: "  padded  ", want: "padded"},
		{name: "empty", in: "", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitiseText(tc.in); got != tc.want {
				t.Errorf("SanitiseText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
