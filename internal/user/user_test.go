package user

import (
	"fmt"
	"testing"
)

func Test_userService_IsAuthorized(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		chatID  string
		want    bool
		wantErr bool
	}{
		{
			name:    "test IsAuthorized false",
			userID:  "0",
			chatID:  "0",
			want:    false,
			wantErr: false,
		},
		{
			name:    "test IsAuthorized true",
			userID:  "1",
			chatID:  "0",
			want:    true,
			wantErr: false,
		},
		{
			name:    "test IsAuthorized error",
			userID:  "2",
			chatID:  "0",
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := userService{
				repo: &userRepoMock{},
			}
			got, gotErr := u.IsAuthorized(tt.userID, tt.chatID)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("IsAuthorized() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("IsAuthorized() succeeded unexpectedly")
			}
			if tt.want != got {
				t.Errorf("IsAuthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userService_GetChatIDs(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		want    []string
		wantErr bool
	}{
		{
			name:    "test GetUserChats",
			userID:  "0",
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "test GetUserChats error",
			userID:  "2",
			want:    []string{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := userService{
				repo: &userRepoMock{},
			}
			_, gotErr := u.GetChatIDs(tt.userID)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("GetChatIDs() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("GetChatIDs() succeeded unexpectedly")
			}
		})
	}
}

type userRepoMock struct{}

func (u *userRepoMock) GetUserChats(userID string) ([]string, error) {
	switch userID {
	case "0":
		return []string{}, nil
	case "1":
		return []string{"0", "1"}, nil
	}

	return []string{}, fmt.Errorf("no such user")
}
