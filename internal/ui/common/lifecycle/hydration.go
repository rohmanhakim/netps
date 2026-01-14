package lifecycle

type HydrationState int

const (
	StateNotAsked HydrationState = iota
	StateHydrating
	StateSuccess
	StateError
)
