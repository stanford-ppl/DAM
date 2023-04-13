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

// type PMU[T datatypes.DAMType] struct {
// 	core.PrimitiveNode[[]T]
// 	MaxSize   int
// 	behavior  PMUBehavior
// 	readData  []PMURead
// 	writeData []PMUWrite
// }

// // Compile-time assertion that PMUs are Contexts
// var _ core.Context = (*PMU[datatypes.DAMType])(nil)

// func (pmu *PMU[T]) Init() {
// 	v := make([]T, pmu.MaxSize)
// 	pmu.State = &v
// }

// func (pmu *PMU[T]) mapAndCheckIndex(index int) int {
// 	if !pmu.behavior.NO_MOD_ADDRESS {
// 		return index % pmu.MaxSize
// 	}
// 	if index >= pmu.MaxSize {
// 		panic(fmt.Sprintf("Out of bounds access at address %d (PMU Size %d)", index, pmu.MaxSize))
// 	}
// 	return index
// }

// func (pmu *PMU[T]) Write(index int, value T) {
// 	(*pmu.State)[pmu.mapAndCheckIndex(index)] = value
// }

// func (pmu *PMU[T]) Read(index int) datatypes.DAMType {
// 	return (*pmu.State)[pmu.mapAndCheckIndex(index)]
// }

// func (pmu *PMU[T]) AddReader(addr core.CommunicationChannel, outputs []core.CommunicationChannel, tp AccessType) {
// 	readData := PMURead{
// 		Type: tp,
// 		Addr: pmu.AddInputChannel(addr),
// 		Outputs: utils.Map(outputs, func(channel core.CommunicationChannel) int {
// 			return pmu.AddInputChannel(channel)
// 		}),
// 	}
// 	pmu.readData = append(pmu.readData, readData)
// }

// func (pmu *PMU[T]) AddWriter(addr core.CommunicationChannel, data core.CommunicationChannel,
// 	enable utils.Option[core.CommunicationChannel], ack []core.CommunicationChannel, tp AccessType,
// ) {
// 	wData := PMUWrite{
// 		Type: tp,
// 		Addr: pmu.AddInputChannel(addr),
// 		Data: pmu.AddInputChannel(data),
// 		Ack: utils.Map(ack, func(channel core.CommunicationChannel) int {
// 			return pmu.AddInputChannel(channel)
// 		}),
// 		Enable: -1,
// 	}
// 	if enable.IsSet() {
// 		wData.Enable = pmu.AddInputChannel(enable.Get())
// 	}
// 	pmu.writeData = append(pmu.writeData, wData)
// }

// func (pmu *PMU[T]) Run() {
// 	// Get the first available set of channels
// 	var channelTimings []big.Int
// 	for _, p := range pmu.readData {
// 		channel := pmu.InputChannels[p.Addr]
// 		// check if channel is alive
// 		channelTimings
// 	}
// }
