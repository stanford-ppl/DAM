package plasticine

import (
	"fmt"
	"sync"

	"github.com/stanford-ppl/DAM/core"
	"github.com/stanford-ppl/DAM/datatypes"
	"github.com/stanford-ppl/DAM/templates/shared/accesstypes"
	"github.com/stanford-ppl/DAM/utils"
)

type PMUPacket[T any] struct {
	Time   core.Time
	Status core.Status
	Data   T
}

func (pkt *PMUPacket[T]) String() string {
	return fmt.Sprintf("PMUPacket(%s, %s, %v)", &pkt.Time, pkt.Status, pkt.Data)
}

func PktLT[T any](p1, p2 PMUPacket[T]) bool {
	cmp := p1.Time.Cmp(&p2.Time)
	if cmp == 0 {
		// In the case that they're at the same time, favor the packet that actually has data.
		if p1.Status == core.Nothing {
			return false
		}
		if p2.Status == core.Nothing {
			return true
		}
	}
	return cmp < 0
}

// Contains the basic metadata for a PMU Read request
type PMURead struct {
	Addr    int
	Outputs []int // Broadcast
	Type    accesstypes.AccessType
}

type PMUReadEntry struct {
	PMURead
	AddrValue datatypes.DAMType
	Time      core.Time
}

func (rentry *PMUReadEntry) String() string {
	return fmt.Sprintf("Read(%v) @ %v", rentry.AddrValue, &rentry.Time)
}

// Write:
// Addr Stream (scalar or vector)
// Data Stream (scalar or vector)
// Enable Stream (scalar or vector), optional
// Ack Stream (scalar, optional)
// Type:
// Scalar addr, Vector data -> vector store
// Scalar addr, Scalar data -> scalar store
// Vector addr, Vector data -> scatter

type PMUWrite struct {
	Addr   int
	Data   int
	Enable int
	Ack    []int // Broadcast
	Type   accesstypes.AccessType
}

// Take advantage of Go's zero-defaults to set all of
// these flags to be false by default.
type PMUBehavior struct {
	// Wrap addresses mod MaxSize
	NO_MOD_ADDRESS bool

	USE_DEFAULT_VALUE bool
}

func broadcastEnable(enable utils.Option[datatypes.DAMType], width int) (result []bool) {
	result = make([]bool, width)
	if !enable.IsSet() {
		utils.FillConst(result, true)
		return
	}
	switch enSig := enable.Get().(type) {
	case datatypes.Bit:
		utils.FillConst(result, enSig.Value)
	case datatypes.Vector[datatypes.Bit]:
		utils.Tabulate(result, func(ind int) bool {
			return enSig.Get(ind).Value
		})
	}
	return
}

type PMU[T datatypes.DAMType] struct {
	core.ChildIDManager
	core.HasParent

	datastore PMUDataStore[T]
	reader    PMUReadPipeline[T]
	writer    PMUWritePipeline[T]
	latency   int64
}

var (
	_ core.Context       = (*PMU[datatypes.DAMType])(nil)
	_ core.ParentContext = (*PMU[datatypes.DAMType])(nil)
)

func MakePMU[T datatypes.DAMType](capacity int64, latency int64, behavior PMUBehavior) (pmu *PMU[T]) {
	pmu = &PMU[T]{
		datastore: PMUDataStore[T]{
			Capacity: capacity,
			Behavior: behavior,
		},
		latency: latency,
	}
	return
}

func (pmu *PMU[T]) AddChild(core.Context) {
	panic("PMUs have automatically managed children!")
}

func (pmu *PMU[T]) BlockUntil(time *core.Time) <-chan *core.Time {
	b1 := pmu.reader.BlockUntil(time)
	b2 := pmu.writer.BlockUntil(time)
	ret := make(chan *core.Time, 1)

	go (func() {
		var v1 *core.Time
		var v2 *core.Time
		select {
		case v1 = <-b1:
			v2 = <-b2
		case v2 = <-b2:
			v1 = <-b1
		}
		res := new(core.Time)
		utils.Min[*core.Time](v1, v2, res)
		ret <- res
	})()

	return ret
}

func (pmu *PMU[T]) Cleanup() {
}

func (*PMU[T]) String() string {
	return fmt.Sprintf("PMU[%s]", utils.TypeString[T]())
}

func (pmu *PMU[T]) Init() {
	pmu.reader.Init()
	pmu.writer.Init()
	pmu.reader.SetParent(pmu)
	pmu.writer.SetParent(pmu)
	pmu.datastore.Init()
}

func (pmu *PMU[T]) Run() {
	var wg sync.WaitGroup
	wg.Add(2)
	go (func() { pmu.reader.Run(); pmu.reader.Cleanup(); wg.Done() })()
	go (func() { pmu.writer.Run(); pmu.writer.Cleanup(); wg.Done() })()
	wg.Wait()
}

func (pmu *PMU[T]) TickLowerBound() (ret *core.Time) {
	t1 := pmu.reader.TickLowerBound()
	t2 := pmu.writer.TickLowerBound()
	ret = new(core.Time)
	utils.Min[*core.Time](t1, t2, ret)
	return
}

func (pmuReadPipeline *PMUReadPipeline[T]) makeReadPacket(read PMURead) (packet PMUPacket[PMURead]) {
	pkt, status := pmuReadPipeline.InputChannel(read.Addr).Peek()
	packet.Time.Set(&pkt.Time)
	packet.Status = status
	packet.Data = read
	return packet
}

func (pmu *PMU[T]) AddWriter(addr *core.CommunicationChannel, data *core.CommunicationChannel,
	enable utils.Option[*core.CommunicationChannel], ack []*core.CommunicationChannel, tp accesstypes.AccessType,
) {
	pmu.writer.AddWriter(addr, data, enable, ack, tp)
}

func (pmu *PMU[T]) AddReader(addr *core.CommunicationChannel, outputs []*core.CommunicationChannel, tp accesstypes.AccessType) {
	pmu.reader.AddReader(addr, outputs, tp)
}

type PMUReadPipeline[T datatypes.DAMType] struct {
	core.LLIOWithTime
	parent *PMU[T]

	readData    []PMURead
	readBacklog *PMUReadEntry
}

func (rp *PMUReadPipeline[T]) ParentContext() core.ParentContext {
	return rp.parent
}

func (rp *PMUReadPipeline[T]) SetParent(p core.ParentContext) {
	rp.parent = p.(*PMU[T])
}

func (rp *PMUReadPipeline[T]) String() string {
	return fmt.Sprintf("ReadPipeline[%s]", utils.TypeString[T]())
}

func (rp *PMUReadPipeline[T]) Init() {
	rp.LowLevelIO.InitWithCtx(rp)
}

func (readPipeline *PMUReadPipeline[T]) AddReader(addr *core.CommunicationChannel, outputs []*core.CommunicationChannel, tp accesstypes.AccessType) {
	readData := PMURead{
		Type: tp,
		Addr: readPipeline.AddInputChannel(addr),
		Outputs: utils.Map(outputs, func(channel *core.CommunicationChannel) int {
			return readPipeline.AddOutputChannel(channel)
		}),
	}
	readPipeline.readData = append(readPipeline.readData, readData)
}

func (rp *PMUReadPipeline[T]) Run() {
	for rp.readTick() {
	}
}

func (pmu *PMUReadPipeline[T]) readTick() bool {
	if pmu.readBacklog != nil {
		outputChannels := pmu.readBacklog.Outputs
		channels := utils.Map(outputChannels, pmu.OutputChannel)
		canWrite := utils.Forall(channels, func(cHandle core.OutputChannel) bool {
			return !cHandle.IsFull()
		})
		if canWrite {
			// Fetch result now
			core.GetLogger(pmu).Sugar().Infof("Reading: %+v", pmu.readBacklog)
			// Wait for the write side to catch up
			<-pmu.parent.writer.BlockUntil(&pmu.readBacklog.Time)
			values := pmu.parent.datastore.HandleRead(pmu.readBacklog.AddrValue, pmu.readBacklog.PMURead, &pmu.readBacklog.Time)
			for _, v := range channels {
				v.Enqueue(core.MakeChannelElement(pmu.TickLowerBound(), values))
			}
			pmu.readBacklog = nil
		} else {
			utils.Foreach(channels, func(chn core.OutputChannel) {
				nextTime := chn.NextTime()
				if nextTime != nil {
					pmu.AdvanceToTime(nextTime)
				}
			})
			pmu.IncrCycles(core.OneTick)
			return true
		}
	}

	// No current packet, so let's get one!

	// Scan all of the relevant channels
	packets := utils.Map(pmu.readData, pmu.makeReadPacket)
	livePackets := utils.Filter(packets, func(pkt PMUPacket[PMURead]) bool { return pkt.Status != core.Closed })
	if utils.IsEmpty(livePackets) {
		return false
	}
	// fmt.Println("Live Packets Read")
	// for _, p := range livePackets {
	// 	fmt.Println("Read", p.String())
	// }
	firstPacket := utils.MinElem(livePackets, PktLT[PMURead])
	// fmt.Println("Selected Read", firstPacket.String())
	readData := firstPacket.Data
	// Skip forward to the packet's time
	addrChan := pmu.InputChannel(readData.Addr)
	addr, addrStatus := addrChan.Dequeue()
	pmu.AdvanceToTime(&addr.Time)
	if addrStatus == core.Nothing {
		pmu.IncrCycles(core.OneTick)
		return true
	}
	// fmt.Println("Addr:", addr.Time.String(), fmt.Sprintf("%T %#v", addr.Data, addr.Data), "Status:", addrStatus)

	extendedRead := new(PMUReadEntry)
	extendedRead.PMURead = readData
	extendedRead.Time.Set(pmu.TickLowerBound())
	extendedRead.Time.Add(&extendedRead.Time, core.NewTime(pmu.parent.latency))
	extendedRead.AddrValue = addr.Data
	pmu.readBacklog = extendedRead
	pmu.IncrCycles(core.OneTick)
	return true
}

var _ core.Context = (*PMUReadPipeline[datatypes.DAMType])(nil)

type PMUWritePipeline[T datatypes.DAMType] struct {
	core.LLIOWithTime
	parent *PMU[T]

	writeData    []PMUWrite
	writeBacklog *PMUWrite
}

var _ core.Context = (*PMUWritePipeline[datatypes.DAMType])(nil)

func (wp *PMUWritePipeline[T]) ParentContext() core.ParentContext {
	return wp.parent
}

func (wp *PMUWritePipeline[T]) SetParent(p core.ParentContext) {
	wp.parent = p.(*PMU[T])
}

func (wp *PMUWritePipeline[T]) String() string {
	return fmt.Sprintf("WritePipeline[%s]", utils.TypeString[T]())
}

func (wp *PMUWritePipeline[T]) Init() {
	wp.LowLevelIO.InitWithCtx(wp)
}

func (wp *PMUWritePipeline[T]) Run() {
	for wp.writeTick() {
	}
}

func (pmuWriter *PMUWritePipeline[T]) AddWriter(addr *core.CommunicationChannel, data *core.CommunicationChannel,
	enable utils.Option[*core.CommunicationChannel], ack []*core.CommunicationChannel, tp accesstypes.AccessType,
) {
	wData := PMUWrite{
		Type: tp,
		Addr: pmuWriter.AddInputChannel(addr),
		Data: pmuWriter.AddInputChannel(data),
		Ack: utils.Map(ack, func(channel *core.CommunicationChannel) int {
			return pmuWriter.AddOutputChannel(channel)
		}),
		Enable: -1,
	}
	if enable.IsSet() {
		wData.Enable = pmuWriter.AddInputChannel(enable.Get())
	}
	pmuWriter.writeData = append(pmuWriter.writeData, wData)
}

func (pmuWriter *PMUWritePipeline[T]) writeTick() bool {
	if pmuWriter.writeBacklog != nil {
		// Try to process the backlog. We're stuck until it gets cleared.
		ackChannels := pmuWriter.writeBacklog.Ack
		channels := utils.Map(ackChannels, pmuWriter.OutputChannel)
		canWrite := utils.Forall(channels, func(cHandle core.OutputChannel) bool {
			return !cHandle.IsFull()
		})
		curTime := pmuWriter.TickLowerBound()
		curTime.Add(curTime, core.NewTime(pmuWriter.parent.latency-1))
		if canWrite {
			for _, v := range channels {
				v.Enqueue(core.MakeChannelElement(curTime, datatypes.Bit{}))
			}
			pmuWriter.writeBacklog = nil
		} else {
			// we can't write yet, so we need to wait a bit
			utils.Foreach(channels, func(chn core.OutputChannel) {
				nextTime := chn.NextTime()
				if nextTime != nil {
					pmuWriter.AdvanceToTime(nextTime)
				}
			})
			pmuWriter.IncrCycles(core.OneTick)
			return true
		}
	}

	// No current packet, so let's get one!

	// Scan all of the relevant channels
	packets := utils.Map(pmuWriter.writeData, pmuWriter.makeWritePacket)
	livePackets := utils.Filter(packets, func(pkt PMUPacket[PMUWrite]) bool { return pkt.Status != core.Closed })
	if utils.IsEmpty(livePackets) {
		return false
	}
	// fmt.Println("Live Packets Write")
	// for _, p := range livePackets {
	// 	fmt.Println("Write", p.Status, p.Time.String(), p.Data)
	// }
	firstPacket := utils.MinElem(livePackets, PktLT[PMUWrite])
	if firstPacket.Status == core.Nothing {
		pmuWriter.AdvanceToTime(&firstPacket.Time)
		pmuWriter.IncrCycles(core.OneTick)
		return true
	}
	writeData := firstPacket.Data
	// Skip forward to the packet's time

	reads := []int{writeData.Addr, writeData.Data}
	if writeData.Enable != -1 {
		reads = append(reads, writeData.Enable)
	}
	// fmt.Println("Dequeuing Writes:", reads)
	dequeuedData := core.DequeueInputChansByID(pmuWriter, reads...)
	// fmt.Println("Write Data", dequeuedData)
	addr := dequeuedData[0]
	data := dequeuedData[1]
	// fmt.Println("Addr:", addr.Data.(datatypes.FixedPoint).ToInt().Int64(), "Data:", data.Data)
	var enable utils.Option[datatypes.DAMType]
	if writeData.Enable != -1 {
		enable = utils.Some(dequeuedData[2].Data)
	}

	writeTime := pmuWriter.TickLowerBound()
	writeTime.Add(writeTime, core.NewTime(pmuWriter.parent.latency-1))
	pmuWriter.parent.datastore.HandleWrite(addr.Data, enable, data.Data, writeData, writeTime)
	pmuWriter.writeBacklog = &writeData
	pmuWriter.IncrCycles(core.OneTick)
	return true
}

func (pmuWritePipeline *PMUWritePipeline[T]) makeWritePacket(write PMUWrite) (packet PMUPacket[PMUWrite]) {
	channelsToCheck := []int{
		write.Addr,
		write.Data,
	}
	if write.Enable != -1 {
		channelsToCheck = append(channelsToCheck, write.Enable)
	}
	packet.Status = core.Ok
	for _, chanID := range channelsToCheck {
		cc := pmuWritePipeline.InputChannel(chanID)
		cE, status := cc.Peek()
		// The packet won't have the data until the max of the times.
		utils.Max[*core.Time](&cE.Time, &packet.Time, &packet.Time)
		if status == core.Nothing {
			packet.Status = core.Nothing
		}
		if status == core.Closed {
			packet.Status = core.Closed
		}
	}
	packet.Data = write
	return packet
}
