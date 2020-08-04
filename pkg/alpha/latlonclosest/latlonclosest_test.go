// Copyright 2019 Google LLC
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

// Package latlonclosest allows finding pairs of close points, given lat lon
// values.  The package is ALPHA and subject to backwards incompatible changes.
// It rounds position to approximiately the nearest meter.
package latlonclosest

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// func TestFloatToCoord(t *testing.T) {
// 	// 24 / 4 = 6, so range is from 0 to 6 fs
// 	require.Equal(t, uint32(0), floatToCoord(-1))
// 	require.Equal(t, uint32(0x3fffff), floatToCoord(-0.5))
// 	require.Equal(t, uint32(0x7fffff), floatToCoord(0))
// 	// 0.5 and 1 are 1 short of correct, but like, close enough.
// 	// b = 1011
// 	require.Equal(t, uint32(0xbffffe), floatToCoord(0.5))
// 	require.Equal(t, uint32(0xfffffe), floatToCoord(1))
// }
