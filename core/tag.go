package core

type TagType[D any, U any] struct {
	ID int
}

type InputTagUpdater[D any, U any] interface {
	CanRun(update U) bool
	Update(update U)
}

type InputTag[D any, U any] struct {
	Tag TagType[D, U] // Points back to the Tag

	State        D
	InputChannel <-chan U

	Updater InputTagUpdater[D, U]
}

type OutputTagPublisher[D any, U any] interface {
	// Writes a value of U to OutputChannel
	Publish(iterator []int)
}
type OutputTag[D any, U any] struct {
	Tag           TagType[D, U]
	OutputChannel chan<- U
	Publisher     OutputTagPublisher[D, U]
}
