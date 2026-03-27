package controller

import "time"

const (
	// ControllerFinalizer is the finalizer used by all controllers in this package.
	ControllerFinalizer    = "governance.platform.io/finalizer"
	DefaultRequeueDuration = 60 * time.Minute
	FailedStatusError      = "failed to update status"
)
