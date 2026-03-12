package collections

import "iter"

func FilterMap[K comparable, V any](m map[K]V, fn func(K, V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range m {
			if fn(k, v) {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}
