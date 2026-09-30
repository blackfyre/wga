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

	for index := 0; index < admissionPublishBudget; index++ {
		if !limiter.Admit("client-a", AdmissionPublish) {
			t.Fatalf("publish %d must be admitted", index)
		}
	}
	if limiter.Admit("client-a", AdmissionPublish) {
		t.Error("publish beyond budget must be rejected")
	}

	limiter.Release("client-a", AdmissionPublish)
	if !limiter.Admit("client-a", AdmissionPublish) {
		t.Error("release must restore a publication slot")
	}

	// Releasing when nothing was charged is a no-op and never goes negative.
	limiter.Release("unknown-client", AdmissionPublish)
	limiter.Release("client-a", AdmissionPublish)
	if !limiter.Admit("client-a", AdmissionPublish) {
		t.Error("over-release must not corrupt the budget")
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
