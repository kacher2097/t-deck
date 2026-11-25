package apperr

import (
	"errors"
	"strings"
	"testing"
)

func TestAppErrorWrapAndIsCode(t *testing.T) {
	base := errors.New("disk full")
	err := IO("Không thể lưu tệp", base)
	if !IsCode(err, CodeIO) {
		t.Fatalf("expected CodeIO, got %v", err)
	}
	if !errors.Is(err, base) {
		t.Fatalf("expected wrapped cause preserved")
	}
	if !strings.Contains(err.Error(), "[IO] Không thể lưu tệp") {
		t.Fatalf("unexpected formatted message: %s", err.Error())
	}
}

func TestRecoverCapturesPanicWithoutCrashing(t *testing.T) {
	var captured any
	run := func() (err error) {
		defer Recover("test-worker", &err, func(_ string, r any, _ string) {
			captured = r
		})
		panic("boom")
	}
	err := run()
	if err == nil || !IsCode(err, CodeInternal) {
		t.Fatalf("expected internal AppError from panic, got %v", err)
	}
	if captured != "boom" {
		t.Fatalf("expected panic value 'boom', got %v", captured)
	}
}
