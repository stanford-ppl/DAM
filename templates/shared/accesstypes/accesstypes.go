package accesstypes

// Read:
// Addr Stream (scalar or vector)
// Output Stream (scalar or vector)
// Type:
// Scalar addr -> vector output (vector load)
// Scalar addr -> scalar output (scalar load)
// Vector addr -> vector output (gather)

// This is effectively a sealed class.
type accessEV struct{}

func (accessEV) accessEVKey() {}

type (
	Scalar struct{ accessEV }
	Vector struct {
		accessEV
		Width int
	}
	Gather  struct{ accessEV }
	Scatter struct{ accessEV }
)

type AccessType interface {
	accessEVKey()
}
