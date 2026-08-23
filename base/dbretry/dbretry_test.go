package dbretry

import (
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestDo_SucceedsImmediatelyWithoutError(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("expected 1 call and no error, got calls=%d err=%v", calls, err)
	}
}

func TestDo_RetriesOnDeadlockThenSucceeds(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 3 {
			return &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected eventual success, got error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls (2 retries), got %d", calls)
	}
}

func TestDo_RetriesOnLockWaitTimeout(t *testing.T) {
	calls := 0
	err := Do(func() error {
		calls++
		if calls < 2 {
			return &mysql.MySQLError{Number: 1205, Message: "Lock wait timeout"}
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("expected retry on 1205 then success, got calls=%d err=%v", calls, err)
	}
}

func TestDo_DoesNotRetryNonDeadlockError(t *testing.T) {
	calls := 0
	sentinel := errors.New("business rule violation")
	err := Do(func() error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error to pass through, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (no retry) for non-deadlock error, got %d", calls)
	}
}

func TestDo_GivesUpAfterMaxAttempts(t *testing.T) {
	calls := 0
	deadlock := &mysql.MySQLError{Number: 1213, Message: "Deadlock found"}
	err := Do(func() error {
		calls++
		return deadlock
	})
	if err != deadlock {
		t.Fatalf("expected the last deadlock error to be returned, got %v", err)
	}
	if calls != maxAttempts {
		t.Fatalf("expected exactly %d attempts before giving up, got %d", maxAttempts, calls)
	}
}
