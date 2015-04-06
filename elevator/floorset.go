package elevator

// Maintains the on/off state of a set of ints.
// Answers lowest() and highest() -- FUTURE: efficiently. Tree?
type FloorSet struct {
	set []bool
}
func newFloorSet(count int) *FloorSet {
	return &FloorSet(new([count]bool))
}
func (fs *FloorSet) set(floor Floor) bool {
	prev := fs.set[floor]
	fs.set[floor] = true
	return prev
}
func (fs *FloorSet) clear(floor Floor) bool {
	prev := fs.set[floor]
	fs.set[floor] = false
	return prev
}
// Return the next stop above floor, else floor.
func (fs *FloorSet) higher(floor Floor) Floor {
	for i := floor + 1; i < len(fs.set); i++ {
		if fs.set[i] {
			return i
		}
	}
	return floor
}
func (fs *FloorSet) lowest() int {
	for i := 0; i < len(fs.set); i++ {
		if fs.set[i] {
			return i
		}
	}
	return -1
}
func (fs *FloorSet) highest() int {
	for i := len(fs.set) - 1; i >= 0; i-- {
		if fs.set[i] {
			return i
		}
	}
	return -1
}
