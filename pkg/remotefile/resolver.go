package remotefile

import (
	"context"
	"fmt"
	"strings"
)

type resolveResult struct {
	url     string
	content []byte
	err     error
}

// TODO: could make this more generic
// since this is shared with the container
// resolver
type Resolver struct {
	jobs  int
	queue chan resolveResult

	ctx context.Context
}

func NewResolver() *Resolver {
	return &Resolver{
		ctx:   context.Background(),
		queue: make(chan resolveResult, 2),
	}
}

func (r *Resolver) Add(url string) {
	client := NewClient()
	r.jobs += 1

	go func() {
		content, err := client.Resolve(url)
		r.queue <- resolveResult{url: url, content: content, err: err}
	}()
}

func (r *Resolver) Finish() ([]Spec, error) {

	resultItems := make([]Spec, 0, r.jobs)
	var errs []string
	for r.jobs > 0 {
		result := <-r.queue
		r.jobs -= 1

		if result.err != nil {
			errs = append(errs, result.err.Error())
			continue
		}

		resultItems = append(resultItems, Spec{
			URL:     result.url,
			Content: result.content,
		})
	}

	if len(errs) > 0 {
		return resultItems, fmt.Errorf("failed to resolve remote files: %s", strings.Join(errs, "; "))
	}

	return resultItems, nil
}
