package degraded

type VerificationState string

const (
	StateUnknown     VerificationState = "unknown"
	StatePartial     VerificationState = "partial"
	StateVerified    VerificationState = "verified"
	StateDenied      VerificationState = "denied"
	StateUnavailable VerificationState = "unavailable"
)
