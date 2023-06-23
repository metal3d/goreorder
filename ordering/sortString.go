package ordering

import "sort"

var _ sort.Interface = (*StingList)(nil)

// StingList is a list of strings that *can* be sorted.
//
// Implement sort.Interface
type StingList []string

// Len returns the length of the list.
func (s *StingList) Len() int {
	return len(*s)
}

// Swap swaps the elements with indexes i and j.
func (s StingList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less reports whether the element with index i should sort before the element with index j.
func (s StingList) Less(i, j int) bool {
	return s[i] < s[j]
}

// Sort sorts the list.
func (s *StingList) Sort() {
	sort.Sort(s)
}

// Add adds a string to the list.
func (s *StingList) Add(str string) {
	*s = append(*s, str)
}
