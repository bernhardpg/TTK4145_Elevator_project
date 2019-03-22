package main

import (
	"./fsm"
	"./elevio"
	"./iolights"
	"./optimalOrderAssigner"
	"./nodeStatesHandler"
	"./network"
	"./consensusModule/hallConsensus"
	"./consensusModule/cabConsensus"
	"fmt"
)


func main() {
	var localID fsm.NodeID = 1;
	numFloors := 4;

	// Init channels
	// -----
	fsmChns := fsm.StateMachineChannels {
		ArrivedAtFloorChan: make(chan int),
	}
	iolightsChns := iolights.LightsChannels {
		TurnOnLightsChan: make(chan elevio.ButtonEvent),
		TurnOffLightsChan: make(chan elevio.ButtonEvent),
		FloorIndicatorChan: make(chan int),
	}
	optimalOrderAssignerChns := optimalOrderAssigner.OptimalOrderAssignerChannels {
		HallOrdersChan: make(chan [][] bool),
		CabOrdersChan: make(chan [] bool),
		NewOrderChan: make(chan elevio.ButtonEvent), // TODO move to consensus module
		CompletedOrderChan: make(chan int),
		LocallyAssignedOrdersChan: make(chan [][] bool, 2),
		// Needs a buffer size bigger than one because the optimalOrderAssigner might send on this channel multiple times before FSM manages to receive!
	}
	nodeStatesHandlerChns := nodeStatesHandler.NodeStatesHandlerChannels {
		LocalNodeStateChan: make(chan fsm.NodeState),
		RemoteNodeStatesChan: make(chan fsm.NodeState),
		AllNodeStatesChan: make(chan map[fsm.NodeID] fsm.NodeState),
	}
	hallConsensusChns := hallConsensus.Channels {
		CompletedOrderChan: make(chan int),
		NewOrderChan: make(chan elevio.ButtonEvent),
	}
	cabConsensusChns := cabConsensus.Channels {
		CompletedOrderChan: make(chan int),
		NewOrderChan: make(chan int),
	}


	elevio.Init("localhost:15657", numFloors);

	// Start modules
	// -----
	go elevio.IOReader(
		numFloors,
		hallConsensusChns.NewOrderChan,
		cabConsensusChns.NewOrderChan,
		fsmChns.ArrivedAtFloorChan,
		iolightsChns.FloorIndicatorChan)

	go fsm.StateMachine(
		localID, numFloors,
		fsmChns.ArrivedAtFloorChan,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan,
		optimalOrderAssignerChns.CompletedHallOrderChan,
		optimalOrderAssignerChns.CompletedCabOrderChan,
		nodeStatesHandlerChns.LocalNodeStateChan)

	go iolights.LightHandler(
		numFloors,
		iolightsChns.TurnOffHallLightChan,
		iolightsChns.TurnOnHallLightChan,
		iolightsChns.TurnOffCabLightChan,
		iolightsChns.TurnOnCabLightChan,
		iolightsChns.FloorIndicatorChan)

	go nodeStatesHandler.NodeStatesHandler(
		localID,
		nodeStatesHandlerChns.LocalNodeStateChan, nodeStatesHandlerChns.RemoteNodeStatesChan,
		nodeStatesHandlerChns.AllNodeStatesChan)

	go optimalOrderAssigner.Assigner(
		localID, numFloors,
		optimalOrderAssignerChns.HallOrdersChan, optimalOrderAssignerChns.CabOrdersChan,
		optimalOrderAssignerChns.LocallyAssignedOrdersChan, optimalOrderAssignerChns.NewOrderChan,
		optimalOrderAssignerChns.CompletedOrderChan,
		nodeStatesHandlerChns.AllNodeStatesChan,
		iolightsChns.TurnOffLightsChan, iolightsChns.TurnOnLightsChan)

	fmt.Println("(main) Started all modules");

	for {
		select {}
	}
}
