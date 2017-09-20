// Copyright (C) 2017 go-nebulas authors
//
// This file is part of the go-nebulas library.
//
// the go-nebulas library is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// the go-nebulas library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with the go-nebulas library.  If not, see <http://www.gnu.org/licenses/>.
//

package trie

import (
	"bytes"
	"errors"
	"github.com/nebulasio/go-nebulas/crypto/hash"
)

// MerkleProof is a path from root to the proved node
// every element in path is the ir for a node
type MerkleProof [][][]byte

// Prove the associated node to the key exists in trie
// if exists, MerkleProof is a complete path from root to the node
// otherwise, MerkleProof is nil
func (t *Trie) Prove(key []byte) (MerkleProof, error) {
	route := keybytesToHex(key)
	rootHash := t.rootHash
	var proof MerkleProof
	for len(route) > 0 {
		// fetch sub-trie root node
		rootNode, err := t.FetchNode(rootHash)
		if err != nil {
			return nil, err
		}
		flag, err := rootNode.Flag()
		if err != nil {
			return nil, err
		}
		switch flag {
		case branch:
			proof = append(proof, rootNode.Val)
			rootHash = rootNode.Val[route[0]]
			route = route[1:]
		case ext:
			path := rootNode.Val[1]
			next := rootNode.Val[2]
			matchLen := prefixLen(path, route)
			if matchLen != len(path) {
				return nil, errors.New("not found")
			}
			proof = append(proof, rootNode.Val)
			rootHash = next
			route = route[matchLen:]
		case leaf:
			path := rootNode.Val[1]
			matchLen := prefixLen(path, route)
			if matchLen != len(path) {
				return nil, errors.New("not found")
			}
			proof = append(proof, rootNode.Val)
			return proof, nil
		default:
			return nil, errors.New("not found")
		}
	}
	return nil, errors.New("not found")
}

// Verify whether the merkle proof from root to the associated node is right
func (t *Trie) Verify(root []byte, key []byte, proof MerkleProof) error {
	route := keybytesToHex(key)
	length := len(proof)
	wantHash := root
	for i := 0; i < length; i++ {
		val := proof[i]
		ir, err := t.serializer.Serialize(val)
		if err != nil {
			return err
		}
		proofHash := hash.Sha3256(ir)
		if !bytes.Equal(wantHash, proofHash) {
			return errors.New("wrong hash")
		}
		switch len(proof[i]) {
		case 16: // Branch Node
			wantHash = val[route[0]]
			route = route[1:]
			break
		case 3: // Extension Node or Leaf Node
			if val[0] == nil || len(val) == 0 {
				return errors.New("nil flag")
			}
			if val[0][0] == byte(ext) {
				extLen := len(val[1])
				if !bytes.Equal(val[1], route[:extLen]) {
					return errors.New("wrong hash")
				}
				wantHash = val[2]
				route = route[extLen:]
				break
			} else if val[0][0] == byte(leaf) {
				if !bytes.Equal(val[1], route) {
					return errors.New("wrong hash")
				}
				return nil
			}
			return errors.New("unknown flag")
		default:
			return errors.New("wrong proof value, expect [][16][]byte or [][3][]byte")
		}

	}
	return nil
}
