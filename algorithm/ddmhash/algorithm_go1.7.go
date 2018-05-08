
// +build !go1.8

package ddmhash

func cacheSize(block uint64) uint64 {

	epoch := int(block / epochLength)
	if epoch < maxEpoch {
		return cacheSizes[epoch]
	}

	panic("fast prime testing unsupported in Go < 1.8")
}

func datasetSize(block uint64) uint64 {

	epoch := int(block / epochLength)
	if epoch < maxEpoch {
		return datasetSizes[epoch]
	}

	panic("fast prime testing unsupported in Go < 1.8")
}
