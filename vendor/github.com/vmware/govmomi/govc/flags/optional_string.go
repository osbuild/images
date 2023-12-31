/*
Copyright (c) 2023-2023 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flags

import (
	"flag"
)

type optionalString struct {
	val **string
}

func (s *optionalString) Set(input string) error {
	*s.val = &input
	return nil
}

func (s *optionalString) Get() interface{} {
	if *s.val == nil {
		return nil
	}
	return **s.val
}

func (s *optionalString) String() string {
	if s.val == nil || *s.val == nil {
		return "<nil>"
	}
	return **s.val
}

// NewOptionalString returns a flag.Value implementation where there is no default value.
// This avoids sending a default value over the wire as using flag.StringVar() would.
func NewOptionalString(v **string) flag.Value {
	return &optionalString{v}
}
