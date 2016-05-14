package minimu9

// DataAvailabilityError is a warning which tells that some data was
// either lost (not read by the user before it was overwritten with a new value),
// or not available yet (the measurement frequency is too low).
type DataAvailabilityError struct {
	NewDataNotAvailable   bool
	NewDataWasOverwritten bool
}

// Error returns human-readable description string for the error.
func (e *DataAvailabilityError) Error() string {
	if e.NewDataNotAvailable {
		return "Warning: there was no new measurement since the previous read."
	}
	if e.NewDataWasOverwritten {
		return "Warning: a new measurement was acquired before the previous was read."
	}
	return "An unknown error has occured. Data may be stale."
}
