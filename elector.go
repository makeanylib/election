package election

import "context"

type Elector interface {
	// Elect informs the elector to start the election process.
	// The elector will call the callback function when it becomes the leader.
	// When the callback function returns, the elector will step down from the leadership.
	// The elector will retry the election process if the callback function returns an error.
	// To stop the election process, the caller should cancel the provided context.
	Elect(
		ctx context.Context,
		fn func(ctx context.Context) error,
	) error
}
