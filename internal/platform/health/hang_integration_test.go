//go:build integration

package health_test

import (
	"testing"
	"time"
)

// TICK-225 throwaway, never merged: a hang the -timeout must catch.
func TestTICK225Hang(t *testing.T) {
	time.Sleep(time.Hour)
}
