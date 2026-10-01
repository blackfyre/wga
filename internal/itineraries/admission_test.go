package itineraries

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestAdmissionLimiterBoundedPerIdentity(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})

	for index := 0; index < admissionDraftBudget; index++ {
		if !limiter.Admit("client-a", AdmissionDraft) {
			t.Fatalf("draft %d must be admitted", index)
		}
	}
	if limiter.Admit("client-a", AdmissionDraft) {
		t.Error("draft beyond budget must be rejected")
	}

	// A different identity has its own budget.
	if !limiter.Admit("client-b", AdmissionDraft) {
		t.Error("a different identity must not share the first identity's budget")
	}

	// Publication and draft budgets are independent.
	for index := 0; index < admissionPublishBudget; index++ {
		if !limiter.Admit("client-a", AdmissionPublish) {
			t.Fatalf("publish %d must be admitted", index)
		}
	}
	if limiter.Admit("client-a", AdmissionPublish) {
		t.Error("publish beyond budget must be rejected")
	}
}

func TestAdmissionLimiterStoresHashesOnly(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})
	identity := "192.0.2.7"

	limiter.Admit(identity, AdmissionDraft)

	for key := range limiter.keys {
		if key == identity {
			t.Error("limiter must not store the raw identity")
		}
		if len(key) != 64 {
			t.Errorf("storage key length = %d, want 64 (hex sha256)", len(key))
		}
	}
}

func TestAdmissionLimiterReleaseRestoresBudget(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})

	reservations := make([]AdmissionReservation, 0, admissionPublishBudget)
	for index := 0; index < admissionPublishBudget; index++ {
		reservation, ok := limiter.Reserve("client-a", AdmissionPublish)
		if !ok {
			t.Fatalf("publish %d must be admitted", index)
		}
		reservations = append(reservations, reservation)
	}
	if limiter.Admit("client-a", AdmissionPublish) {
		t.Error("publish beyond budget must be rejected")
	}

	reservations[0].Release()
	if !limiter.Admit("client-a", AdmissionPublish) {
		t.Error("release must restore a publication slot")
	}

	// Releasing twice, or releasing a zero reservation, is a no-op and never
	// frees a slot another admission still holds.
	reservations[0].Release()
	AdmissionReservation{}.Release()
	if limiter.Admit("client-a", AdmissionPublish) {
		t.Error("repeated release must not free another admission's slot")
	}
}

func TestAdmissionLimiterReleasesTheMatchingReservation(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})
	start := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	now := start
	limiter.now = func() time.Time { return now }

	// An earlier operation reserves first; a later one reserves and succeeds;
	// then the earlier one fails and releases.
	earlier, _ := limiter.Reserve("client-a", AdmissionDraft)
	now = start.Add(30 * time.Minute)
	if _, ok := limiter.Reserve("client-a", AdmissionDraft); !ok {
		t.Fatal("later draft must be admitted")
	}
	earlier.Release()
	for index := 0; index < 2; index++ {
		if !limiter.Admit("client-a", AdmissionDraft) {
			t.Fatalf("draft %d must be admitted after the release", index+1)
		}
	}
	if limiter.Admit("client-a", AdmissionDraft) {
		t.Fatal("budget must be exhausted")
	}

	// Had the later success been released instead, a slot would free when the
	// earlier reservation expires. It must not: all three kept admissions are
	// at 12:30 and remain inside the trailing hour at 13:05.
	now = start.Add(65 * time.Minute)
	if limiter.Admit("client-a", AdmissionDraft) {
		t.Error("the kept admissions must still count inside the trailing hour")
	}
}

func TestAdmissionLimiterWindowRollover(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }

	for index := 0; index < admissionDraftBudget; index++ {
		if !limiter.Admit("client-a", AdmissionDraft) {
			t.Fatalf("draft %d must be admitted", index)
		}
	}
	if limiter.Admit("client-a", AdmissionDraft) {
		t.Error("draft beyond budget must be rejected within the window")
	}

	now = now.Add(admissionWindow)
	if !limiter.Admit("client-a", AdmissionDraft) {
		t.Error("the budget must roll over once the window has elapsed")
	}
}

func TestAdmissionLimiterEnforcesTrailingWindow(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})
	start := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	now := start
	limiter.now = func() time.Time { return now }

	// One draft early in the hour, two just before its end.
	if !limiter.Admit("client-a", AdmissionDraft) {
		t.Fatal("first draft must be admitted")
	}
	now = start.Add(59 * time.Minute)
	for index := 0; index < 2; index++ {
		if !limiter.Admit("client-a", AdmissionDraft) {
			t.Fatalf("draft %d must be admitted", index+2)
		}
	}

	// Once the first draft leaves the trailing hour exactly one slot frees; the
	// two recent drafts still count, so no fixed-bucket reset admits a burst.
	now = start.Add(61 * time.Minute)
	if !limiter.Admit("client-a", AdmissionDraft) {
		t.Fatal("the slot freed by the expired draft must be admitted")
	}
	if limiter.Admit("client-a", AdmissionDraft) {
		t.Error("more than three drafts within a trailing hour must be refused")
	}

	// After the recent drafts expire, their slots free as well.
	now = start.Add(119 * time.Minute)
	for index := 0; index < 2; index++ {
		if !limiter.Admit("client-a", AdmissionDraft) {
			t.Fatalf("draft %d after the recent drafts expired must be admitted", index+1)
		}
	}
	if limiter.Admit("client-a", AdmissionDraft) {
		t.Error("the budget must stay bounded after slots free")
	}
}

func TestAdmissionLimiterBoundedKeys(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})

	for index := 0; index < admissionMaxKeys*2; index++ {
		limiter.Admit(fmt.Sprintf("client-%d", index), AdmissionDraft)
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if len(limiter.keys) > admissionMaxKeys {
		t.Errorf("tracked keys = %d, want at most %d", len(limiter.keys), admissionMaxKeys)
	}
}

func TestAdmissionLimiterAtomicConcurrency(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})

	const workers = 32
	var wg sync.WaitGroup
	admitted := make([]bool, workers)

	for index := 0; index < workers; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			admitted[index] = limiter.Admit("shared-identity", AdmissionPublish)
		}(index)
	}
	wg.Wait()

	successes := 0
	for _, ok := range admitted {
		if ok {
			successes++
		}
	}
	if successes != admissionPublishBudget {
		t.Errorf("concurrent admissions = %d, want exactly %d", successes, admissionPublishBudget)
	}
}

func TestAdmissionLimiterDefaultsRefuseFourthDraft(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{})

	for index := 0; index < 3; index++ {
		if !limiter.Admit("client-a", AdmissionDraft) {
			t.Fatalf("default draft %d must be admitted", index+1)
		}
	}
	if limiter.Admit("client-a", AdmissionDraft) {
		t.Error("a fourth draft within the hour must be refused under the default budget")
	}
}

func TestAdmissionLimiterHonoursConfiguredBudgets(t *testing.T) {
	limiter := NewAdmissionLimiter(AdmissionBudgets{Drafts: 1000, Publishes: 1})

	for index := 0; index < 20; index++ {
		if !limiter.Admit("client-a", AdmissionDraft) {
			t.Fatalf("draft %d must be admitted under a raised budget", index+1)
		}
	}

	if !limiter.Admit("client-a", AdmissionPublish) {
		t.Fatal("first publication must be admitted")
	}
	if limiter.Admit("client-a", AdmissionPublish) {
		t.Error("publication beyond the configured budget must be refused")
	}
}
