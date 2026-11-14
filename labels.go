// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package asmgen

import (
	"strconv"
)

// Labels hands out label names that are unique within one handler or file.
type Labels struct {
	// used records every name already handed out.
	used map[string]struct{}

	// prefix is joined in front of every stem.
	prefix string
}

// NewLabels creates a label allocator whose names start with prefix.
//
// Takes prefix (string) which is usually the handler name.
//
// Returns *Labels ready for use.
func NewLabels(prefix string) *Labels {
	return &Labels{used: make(map[string]struct{}), prefix: prefix}
}

// New returns a fresh label name for the stem.
//
// Takes stem (string) which describes the label's purpose, such as "loop" or "tail".
//
// Returns string which is unique among the names this Labels has returned.
func (l *Labels) New(stem string) string {
	base := l.prefix + "_" + stem
	name := base
	for count := 2; l.isUsed(name); count++ {
		name = base + "_" + strconv.Itoa(count)
	}
	l.used[name] = struct{}{}
	return name
}

// isUsed reports whether name has already been handed out.
//
// Takes name (string) which is the candidate label.
//
// Returns bool which is true when the name is taken.
func (l *Labels) isUsed(name string) bool {
	_, ok := l.used[name]
	return ok
}
