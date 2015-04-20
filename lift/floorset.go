package lift

import (
	"fmt"
)

// Maintains the on/off state of a arr of ints.
// Answers lowest() and highest() -- FUTURE: efficiently. Tree?
type FloorSet struct {
	arr      []bool
	maxFloor Floor
}

func newFloorSet(count int) *FloorSet {
	return &FloorSet{make([]bool, count), Floor(count - 1)}
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

// Return nearest enabled in direction from floor; if none found, return the argument floor.
func (fs *FloorSet) nearest(cur Floor, dir Direction) (Floor, bool) {
	if dir != UP && dir != DOWN {
		panic(fmt.Sprintf("Invalid direction for FloorSet.nearest(): %d", dir))
	}
	maxFloor := fs.maxFloor // All floorsets must have equal length
	for f := cur.next(dir); f >= 0 && f <= maxFloor; f = f.next(dir) {
		if fs.arr[f] {
			return f, true
		}
	}
	return InvalidFloor, false
}

// Find the next Floor (in the direction), among the specified FloorSets.
// If none, return cur.
// Same code as FloorSet.nearest
// All floorsets must have equal length!
func nearestInFloorSets(cur Floor, dir Direction, floorSets ...*FloorSet) (Floor, bool) {
	if dir != UP && dir != DOWN {
		panic(fmt.Sprintf("Invalid direction for nearestInFloorSets: %d", dir))
	}
	if len(floorSets) == 0 {
		return InvalidFloor, false
	}
	maxFloor := floorSets[0].maxFloor // All floorsets must have equal length
	for f := cur.next(dir); f >= 0 && f <= maxFloor; f = f.next(dir) {
		for _, fs := range floorSets {
			if fs.arr[f] {
				return f, true
			}
		}
	}
	return InvalidFloor, false
}

// Return the furthest Floor (including the specified <floor>) in the direction.
func (fs *FloorSet) furthest(floor Floor, dir Direction) (Floor, bool) {
	switch dir {
	case UP:
		highest, ok := fs.highest()
		if ok && highest >= floor {
			return highest, true
		}
	case DOWN:
		lowest, ok := fs.lowest()
		if ok && lowest <= floor {
			return lowest, true
		}
	default:
		panic(fmt.Sprintf("Invalid direction for furthest: %d", dir))
	}

	return InvalidFloor, false
}

const InvalidFloor = -1

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
func (fs *FloorSet) lowest() (Floor, bool) {
	for i := Floor(0); i <= fs.maxFloor; i++ {
		if fs.arr[i] {
			return i, true
		}
	}
	return InvalidFloor, false
}
func (fs *FloorSet) highest() (Floor, bool) {
	for i := fs.maxFloor; i >= 0; i-- {
		if fs.arr[i] {
			return i, true
		}
	}
	return InvalidFloor, false
}
