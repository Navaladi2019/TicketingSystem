package main

import (
	"slices"
	"testing"
	"time"
)

func TestStaleRemovesOnlyOldIntegrationResources(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	old := now.Add(-7 * time.Hour)
	young := now.Add(-30 * time.Minute)
	c := func(name string, created time.Time, project string) Resource {
		return Resource{Kind: "container", Name: name, Created: created, Project: project}
	}
	n := func(name string, created time.Time, project string) Resource {
		return Resource{Kind: "network", Name: name, Created: created, Project: project}
	}
	rs := []Resource{
		n("nava-it-net-252", old, ""),
		c("nava-it-pg-252", old, ""),
		c("nava-it-pg-252-s3", old, ""),
		c("nava-it-pgb-pooled-252", old, ""),
		c("nava-it-pgb-noprep-252", old, ""),
		c("nava-it-health-4120-50123", old, ""),
		c("nava-iso-pg-77", old, ""),
		c("nava-iso-pgb-77", old, ""),
		n("nava-iso-net-77", old, ""),
		// A run in progress.
		c("nava-it-pg-900", young, ""),
		n("nava-it-net-900", young, ""),
		// Compose stacks: the dev stack, ci-isolated projects, one whose id
		// happens to start with "it" or "iso", and its default network.
		c("nava-core-1", old, "nava"),
		c("nava-90-postgres-1", old, "nava-90"),
		c("nava-ci5-kc2612-1862", old, ""),
		c("nava-it-pg-1", old, "nava-it"),
		n("nava-it_default", old, "nava-it"),
		n("nava-iso-net-5", old, "nava-iso"),
		n("nava_default", old, "nava"),
		// Near misses of the integration forms.
		c("nava-it-pg-252-extra", old, ""),
		c("nava-it-postgres-1", old, ""),
		n("nava-it-net-252x", old, ""),
		n("nava-it-pg-252", old, ""),
		c("nava-it-net-252", old, ""),
		c("xnava-it-pg-252", old, ""),
		// docker printed no creation time.
		c("nava-it-pg-1000", time.Time{}, ""),
	}
	var got []string
	for _, r := range stale(rs, now, 6*time.Hour) {
		got = append(got, r.Kind+" "+r.Name)
	}
	want := []string{
		"container nava-it-pg-252",
		"container nava-it-pg-252-s3",
		"container nava-it-pgb-pooled-252",
		"container nava-it-pgb-noprep-252",
		"container nava-it-health-4120-50123",
		"container nava-iso-pg-77",
		"container nava-iso-pgb-77",
		"network nava-it-net-252",
		"network nava-iso-net-77",
	}
	if !slices.Equal(got, want) {
		t.Errorf("stale:\n got %q\nwant %q", got, want)
	}
}

func TestStaleKeepsResourcesAtExactlyTheAge(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	rs := []Resource{{Kind: "network", Name: "nava-it-net-1", Created: now.Add(-6 * time.Hour)}}
	if got := stale(rs, now, 6*time.Hour); len(got) != 0 {
		t.Errorf("stale removed %v at exactly -older", got)
	}
}

func TestParseReadsDockerListings(t *testing.T) {
	// docker ps and docker network ls output, as on Docker Desktop (CRLF on
	// Windows, fractional seconds on networks).
	out := []byte("nava-it-pg-252\t2026-10-27 00:28:26 +0000 UTC\t\r\n" +
		"nava-it_default\t2026-10-26 23:26:28.480651299 +0000 UTC\tnava-it\n\n")
	rs, err := parse("network", out)
	if err != nil {
		t.Fatal(err)
	}
	want := []Resource{
		{Kind: "network", Name: "nava-it-pg-252", Created: time.Date(2026, 10, 27, 0, 28, 26, 0, time.UTC)},
		{Kind: "network", Name: "nava-it_default", Created: time.Date(2026, 10, 26, 23, 26, 28, 480651299, time.UTC), Project: "nava-it"},
	}
	if len(rs) != len(want) {
		t.Fatalf("parse: got %d resources, want %d: %+v", len(rs), len(want), rs)
	}
	for i := range want {
		if rs[i].Name != want[i].Name || !rs[i].Created.Equal(want[i].Created) || rs[i].Project != want[i].Project {
			t.Errorf("parse[%d] = %+v, want %+v", i, rs[i], want[i])
		}
	}
}

func TestParseRejectsMalformedLines(t *testing.T) {
	for _, line := range []string{"nava-it-pg-1\t2026-10-27 00:28:26 +0000 UTC", "nava-it-pg-1\tyesterday\t"} {
		if _, err := parse("container", []byte(line+"\n")); err == nil {
			t.Errorf("parse(%q): want an error", line)
		}
	}
}

func TestGoneRecognisesAResourceRemovedSinceTheListing(t *testing.T) {
	for out, want := range map[string]bool{
		"Error response from daemon: network nava-it-net-19135 not found":                                      true,
		"Error response from daemon: No such container: nava-it-pg-1":                                          true,
		"Error response from daemon: error while removing network: network nava-it-net-1 has active endpoints": false,
		"Cannot connect to the Docker daemon":                                                                  false,
	} {
		if got := gone([]byte(out)); got != want {
			t.Errorf("gone(%q) = %v, want %v", out, got, want)
		}
	}
}
