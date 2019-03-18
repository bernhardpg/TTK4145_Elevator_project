package fsm

import (
	"../elevio"
	"time"
	"../stateHandler"
)

// StateMachineChannels ...
// Channels used for communication with the Elevator FSM
type StateMachineChannels struct {
	NewOrder chan elevio.ButtonEvent
	ArrivedAtFloor chan int 
}

/*// TODO move these data types to correct file!
type ReqState int;
const (
	Unknown ReqState = iota;
	Inactive
	PendingAck
	Confirmed
)

type nodeId int;

type Req struct {
	state ReqState;
	ackBy []nodeId;
}*/

	
// Initialize locally assigned orders matrix
/*var locallyAssignedOrders = make([][]Req, numFloors);
for i := range assignedOrders {
	assignedOrders[i] = make([]Req, numStates);
}

// Initialize all to unknown
for floor := range assignedOrders {
	for orderType := range assignedOrders[floor] {
		assignedOrders[floor][orderType].state = Unknown;
	}
}*/

func calculateNextOrder(currFloor int, currDir stateHandler.OrderDir, assignedOrders [][] bool) (int) {
	numFloors := len(assignedOrders);

	// Find the order closest to floor currFloor, checking only orders in direction currDir first
	if currDir == stateHandler.Up {
		for floor := currFloor + 1; floor <= numFloors - 1; floor++ {
			for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
				if orderType == elevio.BT_HallDown {
					// Skip orders of opposite directon
					continue;
				}
				if assignedOrders[floor][orderType] == true {
					return floor;
				}
			}
		}
		// Check orders of opposite directon last
		for floor := numFloors -1 ; floor >= currFloor + 1; floor-- {
			if assignedOrders[floor][elevio.BT_HallDown] == true {
				return floor;

			}
		}
	} else {
		for floor := currFloor - 1; floor >= 0; floor-- {
			for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
				if orderType == elevio.BT_HallUp {
					// Skip orders of opposite directon
					continue;
				}
				if assignedOrders[floor][orderType] == true {
					return floor;
				}
			}
		}
		// Check orders of opposite directon last
		for floor := 0; floor <= currFloor - 1; floor++ {
			if assignedOrders[floor][elevio.BT_HallUp] == true {
				return floor;
			}
		}
	}

	// Check other directions if no orders are found
	if currDir == stateHandler.Up {
		return calculateNextOrder(currFloor, stateHandler.Down, assignedOrders);
	}
	return calculateNextOrder(currFloor, stateHandler.Up, assignedOrders);
}

func hasOrders(assignedOrders [][] bool) (bool) {
	for floor := 0; floor < len(assignedOrders); floor++ {
		for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
			if assignedOrders[floor][orderType] == true {
				return true;
			}
		}
	}

	return false;
}

func calculateDirection(currFloor int, currOrder int) (stateHandler.OrderDir) {
	if currOrder > currFloor {
		return stateHandler.Up;
	}
	return stateHandler.Down;
}

func clearOrdersAtFloor(currFloor int, assignedOrders [][]bool, TurnOffLights chan<- elevio.ButtonEvent) {
	for orderType := elevio.BT_HallUp; orderType <= elevio.BT_Cab; orderType++ {
		assignedOrders[currFloor][orderType] = false;
		TurnOffLights <- elevio.ButtonEvent{currFloor, elevio.ButtonType(orderType)};
	}
}

func setOrder(buttonPress elevio.ButtonEvent, assignedOrders [][]bool, TurnOnLights chan<- elevio.ButtonEvent) {
	assignedOrders[buttonPress.Floor][buttonPress.Button] = true;
	TurnOnLights <- buttonPress;
}

// StateHandler ...
// GoRoutine for handling the states of a single elevator
func StateMachine(localID stateHandler.NodeID, numFloors int, NewOrder <-chan elevio.ButtonEvent, ArrivedAtFloor <-chan int, TurnOffLights chan<- elevio.ButtonEvent, TurnOnLights chan<- elevio.ButtonEvent,
	HallOrderChan chan<- [][] bool, CabOrderChan chan<- [] bool, LocalElevStateChan chan<- stateHandler.ElevState) {
	// Initialize variables	
	// -----
	currOrder := -1;
	currFloor := -1;
	var currDir stateHandler.OrderDir = stateHandler.Up;
	doorTimer := time.NewTimer(0);

	assignedOrders := make([][]bool, numFloors);
	for i := range assignedOrders {
		assignedOrders[i] = make([]bool, 3);
	}

	for floor := range assignedOrders {
		for orderType := range assignedOrders[floor] {
			assignedOrders[floor][orderType] = false;
		}
	}

	// Transmit empty order matrices
	transmitHallOrders(assignedOrders, HallOrderChan);
	transmitCabOrders(assignedOrders, CabOrderChan);


	// Initialize elevator
	// -----
	state := stateHandler.InitState
	nextState := state
	updateState := false;
	elevio.SetMotorDirection(elevio.MD_Up)

	// State selector
	// -----
	for {
		select {
		
		case <- doorTimer.C:
			// Door has been open for the desired period of time
			if state != stateHandler.InitState {
				elevio.SetDoorOpenLamp(false)
				if hasOrders(assignedOrders) {
					nextState = stateHandler.Moving
				} else {
					nextState = stateHandler.Idle
				}
			}

		case a := <- ArrivedAtFloor:
			currFloor = a;

			// Transmit state each when reached new floor
			transmitState(localID, state, currFloor, currDir, LocalElevStateChan);

			if state == stateHandler.InitState {
				nextState = stateHandler.Idle
			}

			if currFloor == currOrder {
				clearOrdersAtFloor(currFloor, assignedOrders, TurnOffLights);
				transmitHallOrders(assignedOrders, HallOrderChan)
				transmitCabOrders(assignedOrders, CabOrderChan)
				nextState = stateHandler.DoorOpen
			}

		case a := <- NewOrder:
			// TODO to be replaced with channel input from optimal assigner
			
			// Only open door if already on floor (and not stateHandler.Moving)
			if a.Floor == currFloor && state != stateHandler.Moving {
				// Open door without calculating new order
				nextState = stateHandler.DoorOpen
			} else {
				setOrder(a, assignedOrders, TurnOnLights);
				transmitHallOrders(assignedOrders, HallOrderChan);
				transmitCabOrders(assignedOrders, CabOrderChan);
				if state != stateHandler.DoorOpen {
					// Calculate new order
					nextState = stateHandler.Moving
					// Change state from moving to moving
					updateState = true
				}
			}
		}

		// State transition handling
		// -----
		if nextState != state || updateState {
			switch nextState {
				case stateHandler.DoorOpen:
					elevio.SetMotorDirection(elevio.MD_Stop)
					elevio.SetDoorOpenLamp(true)
					doorTimer.Reset(3 * time.Second)

				case stateHandler.Idle:
					elevio.SetMotorDirection(elevio.MD_Stop)

				case stateHandler.Moving:
					currOrder = calculateNextOrder(currFloor, currDir, assignedOrders)
					currDir = calculateDirection(currFloor, currOrder)

					// Set motor direction
					if currDir == stateHandler.Up {
						elevio.SetMotorDirection(elevio.MD_Up)
					} else {
						elevio.SetMotorDirection(elevio.MD_Down)
					}
			}
			// Set new current state
			state = nextState
			// Transmit state each time state is changed
			transmitState(localID, state, currFloor, currDir, LocalElevStateChan)
			updateState = false

		}
	}
}


func transmitState(localID stateHandler.NodeID, currState stateHandler.BehaviourState, currFloor int, currDir stateHandler.OrderDir, LocalElevStateChan chan<- stateHandler.ElevState) {
	currElevState := stateHandler.ElevState {
		ID: localID,
		State: currState,
		Floor: currFloor,
		Dir: currDir, 
	}

	LocalElevStateChan <- currElevState
}

func transmitCabOrders(assignedOrders [][] bool, CabOrderChan chan<- []bool) {
	// Construct hall order matrix
	numFloors := len(assignedOrders);
	cabOrders := make([]bool, numFloors);

	for i := range assignedOrders {
		cabOrders[i] = assignedOrders[i][elevio.BT_Cab];
	}

	CabOrderChan <- cabOrders;
}


func transmitHallOrders(assignedOrders [][] bool, HallOrderChan chan<- [][]bool) {
	// Construct hall order matrix
	numFloors := len(assignedOrders);
	hallOrders := make([][]bool, numFloors);

	for i := range assignedOrders {
		hallOrders[i] = assignedOrders[i][:elevio.BT_Cab];
	}

	HallOrderChan <- hallOrders;
}
