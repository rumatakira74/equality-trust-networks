// Sparse Merkle-дерево состояния глубины 256 (spec/07 7.1.1).
// Индекс листа = адрес кошелька (32 байта); пустое поддерево -> нулевой хеш.
//
// Схема битов: бит 0 — последний бит адреса (LSB байта 31). На итерации lvl
// узлы — корни поддеревьев из 2^lvl листьев; ключ узла = адрес с обнулёнными
// битами 0..lvl-1. Root() и Prove() используют одну схему; VerifyProof
// независим от реализации.
package smt

import (
	"crypto/sha256"
)

const Depth = 256

var zeroHashes [Depth + 1][32]byte

func init() {
	zeroHashes[0] = sha256.Sum256(nil)
	for i := 1; i <= Depth; i++ {
		zeroHashes[i] = hashPair(zeroHashes[i-1], zeroHashes[i-1])
	}
}

func hashPair(l, r [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], l[:])
	copy(buf[32:], r[:])
	return sha256.Sum256(buf[:])
}

func bitOf(idx [32]byte, lvl int) byte {
	return (idx[lvl/8] >> (7 - uint(lvl%8))) & 1
}

// nodeKey: адрес с обнулёнными битами 0..lvl-1 (ключ узла на итерации lvl).
func nodeKey(idx [32]byte, lvl int) [32]byte {
	p := idx
	for i := 0; i < lvl; i++ {
		p[i/8] &^= 1 << (7 - uint(i%8))
	}
	return p
}

func flipBit(idx [32]byte, lvl int) [32]byte {
	p := idx
	p[lvl/8] ^= 1 << (7 - uint(lvl%8))
	return p
}

func clearBit(idx [32]byte, lvl int) [32]byte {
	p := idx
	p[lvl/8] &^= 1 << (7 - uint(lvl%8))
	return p
}

// Tree — разреженное дерево: хранит хеши непустых листьев.
type Tree struct {
	leaves map[[32]byte][32]byte
}

func New() *Tree { return &Tree{leaves: make(map[[32]byte][32]byte)} }

func (t *Tree) SetLeaf(index [32]byte, leafHash [32]byte) { t.leaves[index] = leafHash }

func (t *Tree) Leaf(index [32]byte) ([32]byte, bool) {
	h, ok := t.leaves[index]
	return h, ok
}

func (t *Tree) Len() int { return len(t.leaves) }

// Root вычисляет корень детерминированно (O(n·depth)).
func (t *Tree) Root() [32]byte {
	if len(t.leaves) == 0 {
		return zeroHashes[Depth]
	}
	level := make(map[[32]byte][32]byte, len(t.leaves))
	for idx, lh := range t.leaves {
		level[idx] = lh // nodeKey(idx, 0) == idx
	}
	for lvl := 0; lvl < Depth; lvl++ {
		next := make(map[[32]byte][32]byte, (len(level)+1)/2)
		for pref, h := range level {
			bit := bitOf(pref, lvl)
			sib := flipBit(pref, lvl)
			other := zeroHashes[lvl]
			if h2, ok := level[sib]; ok {
				other = h2
			}
			parent := clearBit(pref, lvl)
			if bit == 0 {
				next[parent] = hashPair(h, other)
			} else {
				next[parent] = hashPair(other, h)
			}
		}
		level = next
	}
	if len(level) == 0 {
		return zeroHashes[Depth]
	}
	return level[[32]byte{}]
}

// Prove строит доказательство включения листа (siblings от бита 0 к биту 255).
// Сложность O(n·depth) — оптимизация на этапе 5.
func (t *Tree) Prove(index [32]byte) (siblings [][32]byte, leaf [32]byte, ok bool) {
	leaf, ok = t.leaves[index]
	if !ok {
		return nil, [32]byte{}, false
	}
	cur := index
	for lvl := 0; lvl < Depth; lvl++ {
		sib := flipBit(cur, lvl)
		sibHash := t.subtreeHash(sib, lvl)
		siblings = append(siblings, sibHash)
		cur = clearBit(cur, lvl)
	}
	return siblings, leaf, true
}

// subtreeHash — хеш узла итерации lvl с ключом pref (поддерево из листьев,
// у которых биты lvl..255 совпадают с pref).
func (t *Tree) subtreeHash(pref [32]byte, lvl int) [32]byte {
	level := make(map[[32]byte][32]byte)
	for idx, lh := range t.leaves {
		if nodeKey(idx, lvl) == pref {
			level[idx] = lh
		}
	}
	if len(level) == 0 {
		return zeroHashes[lvl]
	}
	for i := 0; i < lvl; i++ {
		next := make(map[[32]byte][32]byte)
		for k, h := range level {
			bit := bitOf(k, i)
			sib := flipBit(k, i)
			other := zeroHashes[i]
			if h2, ok := level[sib]; ok {
				other = h2
			}
			parent := clearBit(k, i)
			if bit == 0 {
				next[parent] = hashPair(h, other)
			} else {
				next[parent] = hashPair(other, h)
			}
		}
		level = next
	}
	if len(level) == 0 {
		return zeroHashes[lvl]
	}
	return level[pref]
}

// VerifyProof проверяет доказательство включения против корня.
func VerifyProof(index [32]byte, leaf [32]byte, siblings [][32]byte, root [32]byte) bool {
	if len(siblings) != Depth {
		return false
	}
	cur := index
	h := leaf
	for lvl := 0; lvl < Depth; lvl++ {
		if bitOf(cur, lvl) == 0 {
			h = hashPair(h, siblings[lvl])
		} else {
			h = hashPair(siblings[lvl], h)
		}
		cur = clearBit(cur, lvl)
	}
	return h == root
}
