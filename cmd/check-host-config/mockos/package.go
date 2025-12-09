// Package mockos provides mockable OS functions for testing. Use this package in
// host checks to allow mocking of OS interactions during unit tests. The package
// provides WithXXXFunc functions to set mock implementations in a context.Context,
// and corresponding XXXFunc functions to retrieve them. If no mock is set, the real
// OS functions are used.
package mockos

// Logger is a simplified interface for logging operations.
type Logger interface {
	Print(v ...any)
	Printf(format string, v ...any)
	Println(v ...any)
}
