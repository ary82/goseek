package user

import "slices"

type userService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) UserService {
	return &userService{
		repo: repo,
	}
}

func (u *userService) IsAuthorized(userID string, chatID string) (bool, error) {
	chats, err := u.repo.GetUserChats(userID)
	if err != nil {
		return false, err
	}

	return slices.Contains(chats, chatID), nil
}

func (u *userService) GetChatIDs(userID string) ([]string, error) {
	chats, err := u.repo.GetUserChats(userID)
	return chats, err
}
