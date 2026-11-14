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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLabelsAreUniqueWithinTheAllocator(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		prefix   string
		stems    []string
		expected []string
	}{
		{name: "distinct stems", prefix: "dot", stems: []string{"loop", "tail"}, expected: []string{"dot_loop", "dot_tail"}},
		{name: "repeated stem counts up", prefix: "dot", stems: []string{"loop", "loop", "loop"}, expected: []string{"dot_loop", "dot_loop_2", "dot_loop_3"}},
		{name: "stem colliding with a counted name", prefix: "x", stems: []string{"a", "a", "a_2"}, expected: []string{"x_a", "x_a_2", "x_a_2_2"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			labels := NewLabels(testCase.prefix)
			names := make([]string, 0, len(testCase.stems))
			for _, stem := range testCase.stems {
				names = append(names, labels.New(stem))
			}
			assert.Equal(t, testCase.expected, names)
		})
	}
}
