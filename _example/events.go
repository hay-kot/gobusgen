package events

//go:generate gobusgen generate

// MutationEvent is published when a recipe is created, updated, or deleted.
type MutationEvent struct {
	RecipeID string
	Action   string
}

// UserRegistrationEvent is published when a new user registers.
type UserRegistrationEvent struct {
	UserID string
	Email  string
}

// ShoppingListCleanup is published when a shopping list cleanup is requested.
type ShoppingListCleanup struct {
	ListID string
}

// Events defines all event bus events. The generator reads this map at build time.
var Events = map[string]any{
	"recipe.mutation":       MutationEvent{},
	"user.registration":     UserRegistrationEvent{},
	"shopping_list.cleanup": ShoppingListCleanup{},
}
