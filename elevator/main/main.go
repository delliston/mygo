package main

import (
	"github.com/delliston/mygo/elevator"
)

// This could become a System type
func main() {
	NumFloors := 20	// Floors are numbered from 0
	NumElevators := 3	// TODO: Read these from args

	s := NewSystem(20, 3)

	// Allocate 3 elevators on main floor

	// TODO: Then read commands:
	// - floor [f#] pickup <up|down>
	// - elevator [e#] dropoff [f#]
	// - wait [#steps]
}
