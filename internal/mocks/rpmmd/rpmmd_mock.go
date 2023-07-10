package rpmmd_mock

import (
	dnfjson_mock "github.com/osbuild/images/internal/mocks/dnfjson"
	"github.com/osbuild/images/internal/store"
	"github.com/osbuild/images/internal/worker"
)

type Fixture struct {
	*store.Store
	Workers *worker.Server
	dnfjson_mock.ResponseGenerator
}
