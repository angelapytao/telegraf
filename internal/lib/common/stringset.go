package common

import "sort"

type StringSet map[string]struct{}

func MakeStringSet(strings ...string) StringSet {
	if len(strings) == 0 {
		return nil
	}

	set := StringSet{}
	for _, str := range strings {
		set[str] = struct{}{}
	}
	return set
}

func (set StringSet) Add(s string) {
	set[s] = struct{}{}
}

func (set StringSet) Del(s string) {
	delete(set, s)
}

func (set StringSet) Count() int {
	return len(set)
}

func (set StringSet) Has(s string) (exists bool) {
	if set != nil {
		_, exists = set[s]
	}
	return
}

// Equals compares this StringSet with another StringSet.
func (set StringSet) Equals(anotherSet StringSet) bool {
	if set.Count() != anotherSet.Count() {
		return false
	}

	for k := range set {
		if !anotherSet.Has(k) {
			return false
		}
	}

	return true
}

// ToSlice returns the items in the set as a sorted slice.
func (set StringSet) ToSlice() []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
