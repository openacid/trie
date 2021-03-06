package trie

import "errors"

var (
	// ErrDuplicateKeys indicates two keys are identical.
	ErrDuplicateKeys = errors.New("keys can not be duplicate")

	// ErrKVLenNotMatch means the keys and values to create Trie has different
	// number of elements.
	ErrKVLenNotMatch = errors.New("length of keys and values not equal")

	// ErrKeyOutOfOrder means keys to create Trie are not ascendingly ordered.
	ErrKeyOutOfOrder = errors.New("keys not ascending sorted")
)
