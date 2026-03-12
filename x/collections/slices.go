package collections

func Map[T any, M any](a []T, f func(T) M) []M {
	n := make([]M, len(a))
	for i, e := range a {
		n[i] = f(e)
	}
	return n
}

func Unique[T comparable](input []T) []T {
	seen := make(map[T]bool)
	var result []T

	for _, v := range input {
		if _, ok := seen[v]; !ok {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}
