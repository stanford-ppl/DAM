package plasticine

// import (
// 	"fmt"
// 	"math/big"

// 	"github.com/stanford-ppl/DAM/core"
// 	"github.com/stanford-ppl/DAM/datatypes"
// 	"github.com/stanford-ppl/DAM/utils"
// )

// // Take advantage of Go's zero-defaults to set all of
// // these flags to be false by default.
// type PMUBehavior struct {
// 	// Wrap addresses mod MaxSize
// 	NO_MOD_ADDRESS bool
// }

// // A PMU can have the following IO channels:
// // Read:
// // Addr Stream (scalar or vector)
// // Output Stream (scalar or vector)
// // Type:
// // Scalar addr -> vector output (vector load)
// // Scalar addr -> scalar output (scalar load)
// // Vector addr -> vector output (gather)
// type AccessType uint8

// const (
// 	SCALAR = iota
// 	VECTOR
// 	GATHER_SCATTER
// )

// type PMURead struct {
// 	Addr    int
// 	Outputs []int // Broadcast
// 	Type    AccessType
// }

// // Write:
// // Addr Stream (scalar or vector)
// // Data Stream (scalar or vector)
// // Enable Stream (scalar or vector), optional
// // Ack Stream (scalar, optional)
// // Type:
// // Scalar addr, Vector data -> vector store
// // Scalar addr, Scalar data -> scalar store
// // Vector addr, Vector data -> scatter

// type PMUWrite struct {
// 	Addr   int
// 	Data   int
// 	Enable int
// 	Ack    []int // Broadcast
// 	Type   AccessType
// }

// type PMUData struct {
// 	underlying []datatypes.DAMType
// 	MaxSize    int
// 	behavior   PMUBehavior

// 	readData  []PMURead
// 	writeData []PMUWrite
// }

// type PMU[T datatypes.DAMType] struct {
// 	core.PrimitiveNode[[]T]
// }

// // Compile-time assertion that PMUs are Contexts
// var _ core.Context = (*PMU[datatypes.DAMType])(nil)

// func makePMUData(maxSize int, behavior PMUBehavior) *PMUData {
// 	result := new(PMUData)
// 	result.underlying = make([]datatypes.DAMType, maxSize)
// 	result.MaxSize = maxSize
// 	result.behavior = behavior
// 	return result
// }

// func (data *PMUData) mapAndCheckIndex(index int) int {
// 	if !data.behavior.NO_MOD_ADDRESS {
// 		return index % data.MaxSize
// 	}
// 	if index >= data.MaxSize {
// 		panic(fmt.Sprintf("Out of bounds access at address %d (PMU Size %d)", index, data.MaxSize))
// 	}
// 	return index
// }

// func (data *PMUData) Write(index int, value datatypes.DAMType) {
// 	data.underlying[data.mapAndCheckIndex(index)] = value
// }

// func (data *PMUData) Read(index int) datatypes.DAMType {
// 	return data.underlying[data.mapAndCheckIndex(index)]
// }

// func MakePMU(nElements int, behavioralFlags PMUBehavior) core.Context {
// 	node := core.NewNode()

// 	node.State = makePMUData(nElements, behavioralFlags)

// 	node.Step = func(node *core.Node, ffTime *big.Int) *big.Int {
// 		//
// 		return ffTime
// 	}

// 	return node
// }

// func AddPMURead(node *core.Node, addr core.CommunicationChannel, outputs []core.CommunicationChannel, tp AccessType) {
// 	readData := PMURead{Type: tp}
// 	readData.Addr = node.GetNextInput()
// 	node.SetInputChannel(readData.Addr, addr)

// 	readData.Outputs = make([]int, len(outputs))
// 	for i, cc := range outputs {
// 		outputPort := node.GetNextOutput()
// 		readData.Outputs[i] = outputPort
// 		node.SetOutputChannel(outputPort, cc)
// 	}
// 	node.State.(*PMUData).readData = append(node.State.(*PMUData).readData, readData)
// }

// func AddPMUWrite(node *core.Node, addr core.CommunicationChannel,
// 	data core.CommunicationChannel, enable utils.Option[core.CommunicationChannel], ack []core.CommunicationChannel, tp AccessType,
// ) {
// 	writeData := PMUWrite{Type: tp}
// 	writeData.Addr = node.GetNextInput()
// 	node.SetInputChannel(writeData.Addr, addr)
// 	writeData.Data = node.GetNextInput()
// 	node.SetInputChannel(writeData.Data, data)

// 	if enable.IsSet() {
// 		writeData.Enable = node.GetNextInput()
// 		node.SetInputChannel(writeData.Enable, enable.Get())
// 	} else {
// 		writeData.Enable = -1
// 	}

// 	writeData.Ack = make([]int, len(ack))
// 	for i, cc := range ack {
// 		ackPort := node.GetNextOutput()
// 		writeData.Ack[i] = ackPort
// 		node.SetOutputChannel(ackPort, cc)
// 	}

// 	node.State.(*PMUData).writeData = append(node.State.(*PMUData).writeData, writeData)
// }
