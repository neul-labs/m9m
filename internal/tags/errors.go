package tags

import "errors"

var (
	// ErrTagNotFound is returned when a tag is not found
	ErrTagNotFound = errors.New("tag not found")

	// ErrTagNameRequired is returned when tag name is missing
	ErrTagNameRequired = errors.New("tag name is required")

	// ErrTagNameTooLong is returned when tag name exceeds limit
	ErrTagNameTooLong = errors.New("tag name is too long (max 100 characters)")

	// ErrTagNameExists is returned when a tag with the same name already exists
	ErrTagNameExists = errors.New("tag with this name already exists")

	// ErrTagInUse is returned when trying to delete a tag that's still in use
	ErrTagInUse = errors.New("tag is in use by workflows")
)
