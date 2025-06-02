package user

type UserService interface {
	IsAuthorized(userID string, chatID string) (bool, error)
	GetChatIDs(userID string) ([]string, error)
}

type UserRepository interface {
	GetUserChats(userID string) ([]string, error)
}
