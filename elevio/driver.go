package elevio

import (
	"fmt"
	"net"
	"sync"
	"time"
)

/*

	Disclaimer:
	This driver has mostly been copied from the project description,
	and is not written by authors of this system.

*/

// NumFloors ...
// Used as a global const in the entire project.
// Will be used to initialize size of matrices for the different types defined in 'datatypes.go'
const NumFloors int = 4

const _pollRate = 20 * time.Millisecond

var _initialized = false
var _mtx sync.Mutex
var _conn net.Conn

// MotorDirection ...
// Data type for holding the direction of the elevator motor
type MotorDirection int

const (
	// MD_Up ... Used to Set elevator motor direction to Up
	MD_Up MotorDirection = 1
	// MD_Down ... Used to Set elevator motor direction to Down
	MD_Down = -1
	// MD_Stop ... Used to stop elevator motor
	MD_Stop = 0
)

// ButtonType ...
// Contains the information of the order type of the button pressed
type ButtonType int

const (
	// BT_HallUp ... The button press was for a hall order in the Up direction
	BT_HallUp ButtonType = 0
	// BT_HallDown ... The button press was for a hall order in the Down direction
	BT_HallDown = 1
	// BT_Cab ... The button press was for a cab order
	BT_Cab = 2
)

// ButtonEvent ...
// Is used by the entire project to represent a buttonPress, containing information of both the floor and the order type
type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

// Init ...
// Connects to the elevator hardware or the elevator simulator through TCP
func Init(addr string) {
	if _initialized {
		fmt.Println("Driver already initialized!")
		return
	}
	_mtx = sync.Mutex{}
	var err error
	_conn, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	_initialized = true
}

// SetMotorDirection ...
// Sets the direction of the physical motor
func SetMotorDirection(dir MotorDirection) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{1, byte(dir), 0, 0})
}

// SetButtonLamp ...
// Ignites the lamp on a button
func SetButtonLamp(button ButtonType, floor int, value bool) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{2, byte(button), byte(floor), toByte(value)})
}

// SetFloorIndicator ...
// Ignites the lamp indicating the current floor
func SetFloorIndicator(floor int) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{3, byte(floor), 0, 0})
}

// SetDoorOpenLamp ...
// Ignites the lamp representing that the door is open
func SetDoorOpenLamp(value bool) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{4, toByte(value), 0, 0})
}

// SetStopLamp ...
// Ignites the red 'stop' button
func SetStopLamp(value bool) {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{5, toByte(value), 0, 0})
}

func pollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, NumFloors)
	for {
		time.Sleep(_pollRate)
		for f := 0; f < NumFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				v := getButton(b, f)
				if v != prev[f][b] && v != false {
					receiver <- ButtonEvent{f, ButtonType(b)}
				}
				prev[f][b] = v
			}
		}
	}
}

func pollFloorSensor(receiver chan<- int) {
	prev := -1
	for {
		time.Sleep(_pollRate)
		v := getFloor()
		if v != prev && v != -1 {
			receiver <- v
		}
		prev = v
	}
}

func pollStopButton(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := getStop()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func pollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := getObstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func getButton(button ButtonType, floor int) bool {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{6, byte(button), byte(floor), 0})
	var buf [4]byte
	_conn.Read(buf[:])
	return toBool(buf[1])
}

func getFloor() int {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{7, 0, 0, 0})
	var buf [4]byte
	_conn.Read(buf[:])
	if buf[1] != 0 {
		return int(buf[2])
	}
	return -1
}

func getStop() bool {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{8, 0, 0, 0})
	var buf [4]byte
	_conn.Read(buf[:])
	return toBool(buf[1])
}

func getObstruction() bool {
	_mtx.Lock()
	defer _mtx.Unlock()
	_conn.Write([]byte{9, 0, 0, 0})
	var buf [4]byte
	_conn.Read(buf[:])
	return toBool(buf[1])
}

func toByte(a bool) byte {
	var b byte
	if a {
		b = 1
	}
	return b
}

func toBool(a byte) bool {
	b := false
	if a != 0 {
		b = true
	}
	return b
}
