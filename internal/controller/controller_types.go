package controller

import "time"

const (
	// ControllerFinalizer is the finalizer used by all controllers in this package.
	ControllerFinalizer    = "governance.platform.io/finalizer"
	DefaultRequeueDuration = 60 * time.Minute
	FailedStatusError      = "failed to update status"

	// Annotations for import functionality
	annotationImportID    = "governance.platform.io/import-id"
	annotationImportName  = "governance.platform.io/import-name"
	annotationImportMode  = "governance.platform.io/import-mode"
	importModeObserveOnly = "observe-only"
	importModeOnlyOnce    = "once"
)
