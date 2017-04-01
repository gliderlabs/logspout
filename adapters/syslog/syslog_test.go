package syslog

import (
	"os"
	"strconv"
	"testing"
)

func TestSyslogRetryCount(t *testing.T) {
	setRetryCount()
	if retryCount != defaultRetryCount {
		t.Errorf("expected %v got %v", defaultRetryCount, retryCount)
	}

	newRetryCount := uint(20)
	os.Setenv("RETRY_COUNT", strconv.Itoa(int(newRetryCount)))
	defer os.Unsetenv("RETRY_COUNT")
	setRetryCount()
	if retryCount != newRetryCount {
		t.Errorf("expected %v got %v", newRetryCount, retryCount)
	}
}
