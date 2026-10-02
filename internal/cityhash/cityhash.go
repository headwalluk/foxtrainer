// Package cityhash implements CityHash64 v1.0.x, the version Mozilla bundles to hash Firefox install paths.
//
// It is not CityHash v1.1, which changed the algorithm. Port of Mozilla's
// other-licenses/nsis/Contrib/CityHash/cityhash/city.cpp; see docs/how-it-works.md.
package cityhash

import (
	"encoding/binary"
	"math/bits"
)

const (
	cityK0         uint64 = 0xc3a5c85c97cb3127
	cityK1         uint64 = 0xb492b66fbe98f273
	cityK2         uint64 = 0x9ae16a3b2f90404f
	cityK3         uint64 = 0xc949d7c7509e6557
	cityMultiplier uint64 = 0x9ddfea08eb382d69
)

// Hash64 returns the CityHash64 v1.0 hash of data.
func Hash64(data []byte) uint64 {
	length := len(data)

	var result uint64

	switch {
	case length <= 16:
		result = hashLen0to16(data)
	case length <= 32:
		result = hashLen17to32(data)
	case length <= 64:
		result = hashLen33to64(data)
	default:
		result = hashLongerThan64(data)
	}

	return result
}

// fetch64 reads a little-endian uint64 at offset.
func fetch64(data []byte, offset int) uint64 {
	return binary.LittleEndian.Uint64(data[offset:])
}

// fetch32 reads a little-endian uint32 at offset, widened to uint64.
func fetch32(data []byte, offset int) uint64 {
	return uint64(binary.LittleEndian.Uint32(data[offset:]))
}

// rotate rotates value right by shift bits.
func rotate(value uint64, shift int) uint64 {
	return bits.RotateLeft64(value, -shift)
}

// shiftMix folds the top bits of value into the bottom.
func shiftMix(value uint64) uint64 {
	return value ^ (value >> 47)
}

// hash128to64 reduces a 128-bit value (low, high) to 64 bits.
func hash128to64(low, high uint64) uint64 {
	mixed := (low ^ high) * cityMultiplier
	mixed ^= mixed >> 47
	folded := (high ^ mixed) * cityMultiplier
	folded ^= folded >> 47

	return folded * cityMultiplier
}

// hashLen0to16 hashes inputs of up to 16 bytes.
func hashLen0to16(data []byte) uint64 {
	length := len(data)
	lengthValue := uint64(length)

	var result uint64

	switch {
	case length > 8:
		first := fetch64(data, 0)
		last := fetch64(data, length-8)
		result = hash128to64(first, rotate(last+lengthValue, length)) ^ last
	case length >= 4:
		first := fetch32(data, 0)
		result = hash128to64(lengthValue+(first<<3), fetch32(data, length-4))
	case length > 0:
		byteFirst := uint64(data[0])
		byteMiddle := uint64(data[length>>1])
		byteLast := uint64(data[length-1])
		combinedLow := byteFirst + (byteMiddle << 8)
		combinedHigh := lengthValue + (byteLast << 2)
		result = shiftMix(combinedLow*cityK2^combinedHigh*cityK3) * cityK2
	default:
		result = cityK2
	}

	return result
}

// hashLen17to32 hashes inputs of 17 to 32 bytes.
func hashLen17to32(data []byte) uint64 {
	length := len(data)
	mixA := fetch64(data, 0) * cityK1
	mixB := fetch64(data, 8)
	mixC := fetch64(data, length-8) * cityK2
	mixD := fetch64(data, length-16) * cityK0

	return hash128to64(
		rotate(mixA-mixB, 43)+rotate(mixC, 30)+mixD,
		mixA+rotate(mixB^cityK3, 20)-mixC+uint64(length),
	)
}

// weakHashLen32WithSeeds mixes 32 bytes at offset with two seeds, returning a 128-bit pair.
func weakHashLen32WithSeeds(data []byte, offset int, seedA, seedB uint64) (uint64, uint64) {
	wordW := fetch64(data, offset)
	wordX := fetch64(data, offset+8)
	wordY := fetch64(data, offset+16)
	wordZ := fetch64(data, offset+24)

	seedA += wordW
	seedB = rotate(seedB+seedA+wordZ, 21)
	saved := seedA
	seedA += wordX
	seedA += wordY
	seedB += rotate(seedA, 44)

	return seedA + wordZ, seedB + saved
}

// hashLen33to64 hashes inputs of 33 to 64 bytes.
func hashLen33to64(data []byte) uint64 {
	length := len(data)

	mixZ := fetch64(data, 24)
	mixA := fetch64(data, 0) + (uint64(length)+fetch64(data, length-16))*cityK0
	mixB := rotate(mixA+mixZ, 52)
	mixC := rotate(mixA, 37)
	mixA += fetch64(data, 8)
	mixC += rotate(mixA, 7)
	mixA += fetch64(data, 16)
	firstLow := mixA + mixZ
	firstHigh := mixB + rotate(mixA, 31) + mixC

	mixA = fetch64(data, 16) + fetch64(data, length-32)
	mixZ = fetch64(data, length-8)
	mixB = rotate(mixA+mixZ, 52)
	mixC = rotate(mixA, 37)
	mixA += fetch64(data, length-24)
	mixC += rotate(mixA, 7)
	mixA += fetch64(data, length-16)
	secondLow := mixA + mixZ
	secondHigh := mixB + rotate(mixA, 31) + mixC

	combined := shiftMix((firstLow+secondHigh)*cityK2 + (secondLow+firstHigh)*cityK0)

	return shiftMix(combined*cityK0+firstHigh) * cityK2
}

// hashLongerThan64 hashes inputs longer than 64 bytes in 64-byte chunks.
func hashLongerThan64(data []byte) uint64 {
	length := len(data)

	mixX := fetch64(data, 0)
	mixY := fetch64(data, length-16) ^ cityK1
	mixZ := fetch64(data, length-56) ^ cityK0
	pairVLow, pairVHigh := weakHashLen32WithSeeds(data, length-64, uint64(length), mixY)
	pairWLow, pairWHigh := weakHashLen32WithSeeds(data, length-32, uint64(length)*cityK1, cityK0)
	mixZ += shiftMix(pairVHigh) * cityK1
	mixX = rotate(mixZ+mixX, 39) * cityK1
	mixY = rotate(mixY, 33) * cityK1

	remaining := (length - 1) &^ 63
	offset := 0

	for remaining > 0 {
		mixX = rotate(mixX+mixY+pairVLow+fetch64(data, offset+16), 37) * cityK1
		mixY = rotate(mixY+pairVHigh+fetch64(data, offset+48), 42) * cityK1
		mixX ^= pairWHigh
		mixY ^= pairVLow
		mixZ = rotate(mixZ^pairWLow, 33)
		pairVLow, pairVHigh = weakHashLen32WithSeeds(data, offset, pairVHigh*cityK1, mixX+pairWLow)
		pairWLow, pairWHigh = weakHashLen32WithSeeds(data, offset+32, mixZ+pairWHigh, mixY)
		mixZ, mixX = mixX, mixZ
		offset += 64
		remaining -= 64
	}

	return hash128to64(
		hash128to64(pairVLow, pairWLow)+shiftMix(mixY)*cityK1+mixZ,
		hash128to64(pairVHigh, pairWHigh)+mixX,
	)
}
