package osbuildmonitor

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Status is a high level aggregation of the low-level osbuild monitor
// messages. It is more structured and meant to be used by UI frontends.
//
// this is intentionally minimal at the beginning until we figure
// out the best API, exposing the jsonseq direct feels too messy
// and based on what we learn here we may consider tweaking
// the osubild progress

type Status struct {
	// TODO: this will need to include a "log" or "stage-output"
	// or something so that the full buildlog can be reconstructed
	// from the status streaming

	// Progress contains the current progress.
	Progress *Progress
}

// Progress provides progress information from an osbuild build.
// Each progress can have an arbitrary number of sub-progress information
//
// Note while those can be nested arbitrarly deep in practise
// we are at 2 levels currently:
//  1. overall pipeline progress
//  2. stages inside each pipeline
//
// we might get
//  3. stage progress (e.g. rpm install progress)
//
// in the future
type Progress struct {
	// A human readable message about what is doing on
	Message string
	// The amount of work already done
	Done int
	// The total amount of work for this (sub)progress
	Total int

	SubProgress *Progress
}

// NewStatusScanner returns a StatusScanner that can parse osbuild
// jsonseq monitor status messages
func NewStatusScanner(r io.Reader) *StatusScanner {
	return &StatusScanner{
		scanner:            bufio.NewScanner(r),
		pipelineContextMap: make(map[string]*contextJSON),
	}
}

// StatusScanner scan scan the osbuild jsonseq monitor output
type StatusScanner struct {
	scanner            *bufio.Scanner
	pipelineContextMap map[string]*contextJSON
}

// Status returns a single status struct from the scanner or nil
// if the end of the status reporting is reached.
func (sr *StatusScanner) Status() (*Status, error) {
	if !sr.scanner.Scan() {
		return nil, sr.scanner.Err()
	}

	var status statusJSON
	line := sr.scanner.Bytes()
	line = bytes.Trim(line, "\x1e")
	if err := json.Unmarshal(line, &status); err != nil {
		return nil, fmt.Errorf("cannto scan line %q: %w", line, err)
	}
	// keep track of the context
	// XXX: needs a test
	id := status.Context.Pipeline.ID
	pipelineContext := sr.pipelineContextMap[id]
	if pipelineContext == nil {
		sr.pipelineContextMap[id] = &status.Context
	}

	st := &Status{
		Progress: &Progress{
			Done:  status.Progress.Done,
			Total: status.Progress.Total,
		},
	}
	// add subprogress
	prog := st.Progress
	for subProg := status.Progress.SubProgress; subProg != nil; subProg = subProg.SubProgress {
		prog.SubProgress = &Progress{
			Done:  subProg.Done,
			Total: subProg.Total,
		}
		prog = prog.SubProgress
	}

	return st, nil
}

// statusJSON is a single status entry from the osbuild monitor
type statusJSON struct {
	Context  contextJSON  `json:"context"`
	Progress progressJSON `json:"progress"`
	// Add "Result" here once
	// https://github.com/osbuild/osbuild/pull/1831 is merged

	Message string `json:"message"`
}

// contextJSON is the context for which a status is given. Once a context
// was sent to the user from then on it is only referenced by the ID
type contextJSON struct {
	Origin   string `json:"origin"`
	Pipeline struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Stage struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"stage"`
	} `json:"pipeline"`
}

// progress is the progress information associcated with a given status.
// The details about nesting are the same as for "Progress" above.
type progressJSON struct {
	Name  string `json:"name"`
	Total int    `json:"total"`
	Done  int    `json:"done"`

	SubProgress *progressJSON `json:"progress"`
}
