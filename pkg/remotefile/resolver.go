package remotefile

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
)

type resolveResult struct {
	url     string
	content []byte
	err     error
}

// TODO: could make this more generic since this is shared with the container
// resolver
type Resolver struct {
	queue chan resolveResult
	wg    sync.WaitGroup
	ctx   context.Context
}

func NewResolver(ctx context.Context) *Resolver {
	return &Resolver{
		queue: make(chan resolveResult, 2),
		wg:    sync.WaitGroup{},
		ctx:   ctx,
	}
}

// Add a URL to the resolver queue. When called after Finish was called,
// it may panic.
func (r *Resolver) Add(url string) {
	r.wg.Add(1)
	client := NewClient()

	go func() {
		defer r.wg.Done()

		content, err := client.Resolve(r.ctx, url)
		r.queue <- resolveResult{url: url, content: content, err: err}
	}()
}

// Finish starts collecting of results and returns them. No further calls to Add
// are allowed after this call. It blocks until all results are collected.
func (r *Resolver) Finish() ([]Spec, error) {
	go func() {
		r.wg.Wait()
		close(r.queue)
	}()

	var resultItems []Spec
	var errs []string
	for result := range r.queue {
		if result.err == nil {
			resultItems = append(resultItems, Spec{URL: result.url, Content: result.content})
		} else {
			errs = append(errs, result.err.Error())
		}
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return resultItems, fmt.Errorf("failed to resolve remote files: %s", strings.Join(slices.Compact(errs), "; "))
	}

	return resultItems, nil
}
