package domain

import "errors"

var (
	ErrTitleRequired    = errors.New("title is required")
	ErrFileRequired     = errors.New("file is required")
	ErrCategoryRequired  = errors.New("category is required")
	ErrCategoryCreate   = errors.New("failed to create category")
	ErrCategoryFind     = errors.New("failed to find category")
	ErrUploadFailed     = errors.New("failed to upload file")
	ErrInternal         = errors.New("internal server error")
)
