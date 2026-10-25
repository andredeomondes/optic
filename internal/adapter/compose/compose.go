// Package compose merges several domain.Discoverer sources into one list.
package compose

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"optic/internal/domain"
)

// Discoverer runs every source in parallel and merges the results by IP.
type Discoverer struct {
	Sources []domain.Discoverer
}

// Discover fails only when every source fails; partial results are kept.
func (d Discoverer) Discover(ctx context.Context, timeout time.Duration) ([]domain.TV, error) {
	results := make([][]domain.TV, len(d.Sources))
	errs := make([]error, len(d.Sources))
	var wg sync.WaitGroup
	for i, src := range d.Sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], errs[i] = src.Discover(ctx, timeout)
		}()
	}
	wg.Wait()

	failed := 0
	for _, err := range errs {
		if err != nil {
			failed++
		}
	}
	if len(d.Sources) > 0 && failed == len(d.Sources) {
		return nil, errors.Join(errs...)
	}

	byIP := map[string]domain.TV{}
	for _, tvs := range results {
		for _, tv := range tvs {
			if cur, ok := byIP[tv.IP]; !ok || prefer(tv, cur) {
				byIP[tv.IP] = tv
			}
		}
	}
	out := make([]domain.TV, 0, len(byIP))
	for _, tv := range byIP {
		out = append(out, tv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].IP < out[j].IP })
	return out, nil
}

// prefer reports whether candidate should replace cur for the same IP:
// Android TV wins, otherwise the first entry with a name.
func prefer(candidate, cur domain.TV) bool {
	if cur.Platform == domain.PlatformAndroidTV {
		return false
	}
	if candidate.Platform == domain.PlatformAndroidTV {
		return true
	}
	return cur.Name == "" && candidate.Name != ""
}
