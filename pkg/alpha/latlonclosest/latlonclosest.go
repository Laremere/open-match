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
	"math"
)

// There are a few critical optimizations that this package can make:
// Firstly, we can ignore great circle distance, and instead directly use cord
// distance between points. This is because when comparing the distance
// between one lat lon and two others, the ording of closest neighbors will
// be correct using either method, though the distances are incorrectly.
// Secondly, we don't need to find the actual distance.  Distance between two
// 3d points is sqrt(dx^2 + dy^2 + dz^2).  Since this is positive, dist[0] <
// dist[1] will have the same result as dist[0]^2 < dist[1]^2.  With this, we
// can cancel the sqrt out and arrive at the same answer using only addition
// and multiplication.
// Thirdly, we can ignore the diameter of the real Earch, as scaling doesn't
// change relative order of distances.
// Finally, we can avoid floating point errors by converting to integers.  If M
// is the largest coordinate value, and -M the smallest, the farthest distance
// between two points is
// d = (2m)^2 + (2m)^2 + (2m)^2
// d = 3 * (2m)^2
// d/3 = (2m)^2
// sqrt(d/3) = 2m
// sqrt(d/3)/2 = m
// To avoid int64 overflow, we want d < 2^63
// sqrt((2^63)/3)/2 = m ~= 876706528 ~= 8.76 * 10^8
//
// The diameter of the earth in meters is only 1.3 * 10^7.
// Given that matching people on a sub-meter basis is near meaningless, it's
// resonable to say that we can ignore the fine detail. log2(1.3 * 10^7) ~= 23.6.
// So with only 24 bits, we have more than enough detail on the
// position.
//
// This is useful, because octrees have a pathological problem if too many
// values are at precisly the same position (say, null island (look it up)),
// then it may trigger creating infinitely deep tree.
// Instead, we'll say the max depth is all of the values being at the same
// (roughly) position, and that any query which hits that can choose whichever
// values it wants and still be technically correct.

// I'm over engineering this.

// maxValues is how many values can be in a node before splitting.
const maxValues = 10

const bits = 24

type Q struct {
	root *octree
}

func New() *Q {
	return &Q{
		root: &octree{},
	}
}

// Add includes a value for future query calls.
func (q *Q) Add(id string, lat, lon float64) {
	x, y, z := llToXyz(lat, lon)
	v := &value{x, y, z, id}
	q.root.add(v, 0)
}

// Query returns count closest ids to the provided lat lon.  If you're querying
// for an id which was previously added, it's likely to be the first value,
// however it may not be if multiple values have the same coordinates.
func (q *Q) Query(lat, lon float64, count int) []string {
	return nil //todo
}

type octree struct {
	xMin, yMin, zMin uint32
	d                uint32
	// xMax = xMin + d, etc.

	nodes  [8]*octree
	values []*value
}

type value struct {
	x, y, z uint32
	id      string
}

func (o *octree) add(v *value, depth int) {
	if o.nodes[0] == nil {

	} else {

	}
}

func distSquared(v1, v2 *value) uint64 {
	dx := uint64(v1.x - v2.x)
	dy := uint64(v1.y - v2.y)
	dz := uint64(v1.z - v2.z)
	return dx*dx + dy*dy + dz*dz
}

func llToXyz(lat, lon float64) (x, y, z uint32) {
	// Convert to radians
	lat *= math.Pi / 90
	lon *= math.Pi / 90

	latCos := math.Cos(lat)
	latSin := math.Sin(lat)
	lonCos := math.Cos(lon)
	lonSin := math.Sin(lon)

	x = floatToCoord(latCos * lonCos)
	y = floatToCoord(latCos * lonSin)
	z = floatToCoord(latSin)

	return x, y, z
}

// floatToCoord takes a value from -1 to 1, and returns between 0 (inclusive)
// and 1<<bits (exclusive)
func floatToCoord(f float64) uint32 {
	f += 1
	f *= float64((1 << (bits - 1)) - 1)
	v := uint32(f)
	if v < 0 {
		return 0
	}
	if v >= 1<<bits {
		return 1 << bits
	}
	return v
}
