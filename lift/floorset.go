package lift

import "fmt"

// Maintains the on/off state of a arr of ints.
// Answers lowest() and highest() -- FUTURE: efficiently. Tree?
type FloorSet struct {
	arr []bool
}
func newFloorSet(count int) *FloorSet {
	return &FloorSet{make([]bool, count)}
}
func (fs *FloorSet) set(floor Floor) bool {
	prev := fs.arr[floor]
	fs.arr[floor] = true
	return prev
}
func (fs *FloorSet) clear(floor Floor) bool {
	prev := fs.arr[floor]
	fs.arr[floor] = false
	return prev
}
// Return next set floor in direction
func (fs *FloorSet) next(floor Floor, dir Direction) Floor {
	if dir != UP && dir != DOWN {
		panic(fmt.Sprintf("Invalid direction for next: %d", dir))
	}
	for f := floor.next(dir); f >= 0 && f < Floor(len(fs.arr)); f = f.next(dir) {
		if fs.arr[f] {
			return f
		}
	}
	return floor
}
//
//// Return the next stop above floor, else floor.
//func (fs *FloorSet) higher(floor Floor) Floor {
//	for i := floor + 1; i < len(fs.arr); i++ {
//		if fs.arr[i] {
//			return i
//		}
//	}
//	return floor
//}
func (fs *FloorSet) lowest() int {
	for i := 0; i < len(fs.arr); i++ {
		if fs.arr[i] {
			return i
		}
	}
	return -1
}
func (fs *FloorSet) highest() int {
	for i := len(fs.arr) - 1; i >= 0; i-- {
		if fs.arr[i] {
			return i
		}
	}
	return -1
}
