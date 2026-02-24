package whoop

// Scope represents an OAuth2 scope required to access specific WHOOP API endpoints.
type Scope string

const (
	// ScopeReadRecovery allows reading the user's recovery data.
	ScopeReadRecovery Scope = "read:recovery"

	// ScopeReadCycles allows reading the user's physiological cycles.
	ScopeReadCycles Scope = "read:cycles"

	// ScopeReadSleep allows reading the user's sleep data.
	ScopeReadSleep Scope = "read:sleep"

	// ScopeReadWorkout allows reading the user's workout data.
	ScopeReadWorkout Scope = "read:workout"

	// ScopeReadProfile allows reading the user's basic profile.
	ScopeReadProfile Scope = "read:profile"

	// ScopeReadBodyMeasurement allows reading the user's body measurements.
	ScopeReadBodyMeasurement Scope = "read:body_measurement"
)
