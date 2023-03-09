package core

type TagType[D any, U any] struct {
	TagID int
}

type InputTagUpdater[D any, U any] interface {
	CanRun(update U) bool
	Update(state D, update U) D
}

type InputTag[D any, U any] struct {
	Tag       TagType[D, U] // Points back to the Tag
	InputPort Port

	State D

	Updater InputTagUpdater[D, U]
}

type OutputTagPublisher[D any, U any] interface {
	// Writes a value of U to OutputChannel
	Publish(iterator []int) U
}

type OutputTag[D any, U any] struct {
	Tag        TagType[D, U]
	OutputPort Port
	Publisher  OutputTagPublisher[D, U]
}
