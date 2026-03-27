package queue

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"thing-index/internal/domain"
)

func TestClassifyDocumentIndexFailureMarksPermanentErrors(t *testing.T) {
	t.Parallel()

	cases := []error{
		fmtError(domain.ErrDocumentSourceMissing),
		fmtError(domain.ErrDocumentParsingFailed),
	}

	for _, err := range cases {
		err := err
		t.Run(err.Error(), func(t *testing.T) {
			t.Parallel()

			disposition := classifyDocumentIndexFailure(err, 0, 3, 30*time.Second)
			if disposition.Action != documentIndexFailurePermanent {
				t.Fatalf("expected permanent failure action, got %#v", disposition)
			}
			if disposition.NextRetryAttempt != 0 {
				t.Fatalf("expected no retry attempt for permanent failure, got %#v", disposition)
			}
		})
	}
}

func TestClassifyDocumentIndexFailureSchedulesRetryWhenRetriesRemain(t *testing.T) {
	t.Parallel()

	disposition := classifyDocumentIndexFailure(errors.New("temporary network failure"), 1, 3, 45*time.Second)
	if disposition.Action != documentIndexFailureRetry {
		t.Fatalf("expected retry action, got %#v", disposition)
	}
	if disposition.NextRetryAttempt != 2 {
		t.Fatalf("expected next retry attempt 2, got %#v", disposition)
	}
	if disposition.RetryDelay != 45*time.Second {
		t.Fatalf("expected retry delay to be preserved, got %#v", disposition)
	}
}

func TestClassifyDocumentIndexFailureMovesToDLQWhenRetriesAreExhausted(t *testing.T) {
	t.Parallel()

	disposition := classifyDocumentIndexFailure(errors.New("persistent infra failure"), 3, 3, 30*time.Second)
	if disposition.Action != documentIndexFailureDLQ {
		t.Fatalf("expected dlq action, got %#v", disposition)
	}
	if disposition.NextRetryAttempt != 0 {
		t.Fatalf("expected no next retry attempt, got %#v", disposition)
	}
}

func fmtError(base error) error {
	return fmt.Errorf("%w: wrapped", base)
}
