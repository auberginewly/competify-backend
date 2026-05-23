package agent

import (
	"hash/fnv"
	"strings"
)

// simhash computes a 64-bit SimHash signature for a text.
// It tokenises by whitespace, hashes each token with FNV-1a,
// and accumulates bit-wise votes to produce the final signature.
func simhash(text string) uint64 {
	var vec [64]int
	for _, w := range strings.Fields(strings.ToLower(text)) {
		h := fnvHash(w)
		for i := 0; i < 64; i++ {
			if h&(1<<i) != 0 {
				vec[i]++
			} else {
				vec[i]--
			}
		}
	}
	var sig uint64
	for i := 0; i < 64; i++ {
		if vec[i] > 0 {
			sig |= 1 << i
		}
	}
	return sig
}

// hammingDistance returns the number of differing bits between two 64-bit hashes.
func hammingDistance(a, b uint64) int {
	x := a ^ b
	var count int
	for x != 0 {
		count++
		x &= x - 1
	}
	return count
}

func fnvHash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}
