package usecase

import "errors"

var (
	ErrShipperDocsRequired      = errors.New("shipper: id document and portrait photos are required")
	ErrShipperAlreadyApplied    = errors.New("shipper: an application already exists for this user")
	ErrShipperNotFound          = errors.New("shipper: profile not found")
	ErrShipperAlreadyProcessed  = errors.New("shipper: application already processed")
	ErrShipperRoleNotConfigured = errors.New("shipper: SHIPPER role not configured")
	ErrShipperUploadFailed      = errors.New("shipper: document upload failed")
)
