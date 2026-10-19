package smt

import (
	"crypto/sha256"
	"testing"
)

func TestSmokeRootProveVerify(t *testing.T) {
	tr := New()
	idxs := [][32]byte{sha256.Sum256([]byte("a")), sha256.Sum256([]byte("b")), sha256.Sum256([]byte("c"))}
	for i, idx := range idxs {
		tr.SetLeaf(idx, sha256.Sum256([]byte{byte(i)}))
	}
	root := tr.Root()
	if root == zeroHashes[Depth] {
		t.Fatal("root не должен быть нулевым")
	}
	for i, idx := range idxs {
		sibs, leaf, ok := tr.Prove(idx)
		if !ok || leaf != sha256.Sum256([]byte{byte(i)}) {
			t.Fatal("prove не нашёл лист")
		}
		if !VerifyProof(idx, leaf, sibs, root) {
			t.Fatal("verify не сошёлся")
		}
	}
	// пустое дерево
	if New().Root() != zeroHashes[Depth] {
		t.Fatal("пустое дерево")
	}
}
