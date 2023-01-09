package utils

// SlicePointersToValues takes a slice of pointers and returns a slice of values de-referenced from them.
func SlicePointersToValues[T any](x []*T) []T {
	r := make([]T, len(x))
	for i := 0; i < len(x); i++ {
		r[i] = *x[i]
	}
	return r
}

// SliceValuesToPointers takes a slice of values and returns a slice of pointers to them.
func SliceValuesToPointers[T any](x []T) []*T {
	r := make([]*T, len(x))
	for i := 0; i < len(x); i++ {
		r[i] = &x[i]
	}
	return r
}

// SliceSelect provides a way of querying a specific element from a slice's elements into a slice of its own.
func SliceSelect[T any, K any](x []T, f func(x T) K) []K {
	r := make([]K, len(x))
	for i := 0; i < len(x); i++ {
		r[i] = f(x[i])
	}
	return r
}

// SliceWhere provides a way of querying specific elements which fit some criteria into a new slice.
func SliceWhere[T any](x []T, f func(x T) bool) []T {
	r := make([]T, 0)
	for i := 0; i < len(x); i++ {
		if f(x[i]) {
			r = append(r, x[i])
		}
	}
	return r
}

// Contains checks if a given element e is contained in a slice of elements s. The function takes a slice of elements
// of type T, which must be comparable, and an element of the same type T. The function returns a boolean value
// indicating whether the element e is contained in the slice s.
func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
