// Copyright 2017 The go-ddmchain Authors
// This file is part of the go-ddmchain library.
//
// The go-ddmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ddmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ddmchain library. If not, see <http://www.gnu.org/licenses/>.

// +build !go1.8

package ddmhash

// cacheSize calculates and returns the size of the ddmhash verification cache that
// belongs to a certain block number. The cache size grows linearly, however, we
// always take the highest prime below the linearly growing threshold in order to
// reduce the risk of accidental regularities leading to cyclic behavior.
func cacheSize(block uint64) uint64 {
	// If we have a pre-generated value, use that
	epoch := int(block / epochLength)
	if epoch < maxEpoch {
		return cacheSizes[epoch]
	}
	// We don't have a way to verify primes fast before Go 1.8
	panic("fast prime testing unsupported in Go < 1.8")
}

// datasetSize calculates and returns the size of the ddmhash mining dataset that
// belongs to a certain block number. The dataset size grows linearly, however, we
// always take the highest prime below the linearly growing threshold in order to
// reduce the risk of accidental regularities leading to cyclic behavior.
func datasetSize(block uint64) uint64 {
	// If we have a pre-generated value, use that
	epoch := int(block / epochLength)
	if epoch < maxEpoch {
		return datasetSizes[epoch]
	}
	// We don't have a way to verify primes fast before Go 1.8
	panic("fast prime testing unsupported in Go < 1.8")
}