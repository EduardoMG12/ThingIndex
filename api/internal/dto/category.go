package dto

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
}

type UpdateCollectionCategoryRequest struct {
	Category *string `json:"category" binding:"omitempty,max=255"`
}
