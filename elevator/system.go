package elevator

// The System provisions the elevators (TODO: structs or channels)
// plus the records of any pickup requests (up or down) at each floor.
type System struct {
	elevators []Elevator
	pickupsUp *FloorSet // Floors which have outstanding UP requests are true
	pickupsDown *FloorSet // Floors which have outstanding DOWN requests are true
	chPassengers chan Passenger
	pickupsToPassengers map[Pickup][]Passenger
}
func NewSystem(numFloors, numElevators int) *System {
	elevators := make([]Elevator, numElevators)
	for i := 0; i < numElevators; i++ {
		elevators[i] = NewElevator(numFloors)
	}
	return &System{ elevators, newFloorSet(numFloors), newFloorSet(numFloors), make(chan Pickup), make(map[Pickup][]Passenger)}
}

func (s *System) PickMeUp(floor Floor, dir Direction) chan<- Dropoff {

}

func (s *System) pickups(dir Direction) *FloorSet {
	switch dir {
	case UP:
		return s.pickupsUp
	case DOWN:
		return s.pickupsDown
	default:
		panic("Direction must be up or down, found %d", dir)
	}
}

// Needed because GoLang lacks generics. In Java, would be Map<Pickup, Set<Passenger>>
func (s *System) addPassenger(pass Passenger) {
	arr := s.pickupsToPassengers[pass.pickup]
	if arr == nil {
		arr = make([]Passenger, 1)
	}
	arr = arr.append(pass)
	s.pickupsToPassengers[pass.pickup] = arr
}

func (s *System) mainLoop() {
	// Assumptions: all elevators are at floor 0, and all buttons are cleared
	pickupsToPassL := make(map[Pickup][]Passenger)
	for {
		select {
		case pass := <- s.chPassengers:
			// New passenger.
			s.addPassenger(pass)
			if ! s.pickups(pass.pickup.dir).set(pass.pickup.floor) {
				// Find a suitable elevator
			}

		}
	}
}

