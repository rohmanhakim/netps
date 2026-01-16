package hydration

func expectedWouldChange(
	oldState HydrationState,
	oldErr error,
	newState HydrationState,
	newErr error,
) bool {
	// property 1: state change always matters
	if oldState != newState {
		return true
	}

	// property 2: error identity only matters in StateError
	if oldState == StateError && newState == StateError {
		return oldErr != newErr
	}

	// property 3: otherwise, nothing changed
	return false
}

func errLabel(err error) string {
	if err == nil {
		return "nil"
	}
	return err.Error()
}
