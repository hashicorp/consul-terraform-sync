// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"flag"
	"strings"
)

var _ flag.Value = (*FlagAppendSliceValue)(nil)

// FlagAppendSliceValue implements the flag.Value interface and allows multiple
// calls to the same variable to append a list.
type FlagAppendSliceValue []string

func (s *FlagAppendSliceValue) String() string {
	return strings.Join(*s, ",")
}

func (s *FlagAppendSliceValue) Set(value string) error {
	if *s == nil {
		*s = make([]string, 0, 1)
	}

	*s = append(*s, value)
	return nil
}
