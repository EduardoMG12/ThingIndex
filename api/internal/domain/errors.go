package domain

import "errors"

var (
	ErrTitleRequired     = errors.New("title is required")
	ErrFileRequired      = errors.New("file is required")
	ErrCategoryRequired  = errors.New("category is required")
	ErrCategoryCreate    = errors.New("failed to create category")
	ErrCategoryFind      = errors.New("failed to find category")
	ErrCategoryDuplicate = errors.New("category already exists")
	ErrCategoryNotFound  = errors.New("category not found")
	ErrCategoryUpdate    = errors.New("failed to update category")
	ErrUploadFailed      = errors.New("failed to upload file")
	ErrInternal          = errors.New("internal server error")
)
