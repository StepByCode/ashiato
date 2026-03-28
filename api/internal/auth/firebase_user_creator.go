package auth

import (
	"context"

	fbauth "firebase.google.com/go/v4/auth"
)

// FirebaseUserCreatorAdapter wraps the Firebase Auth client to create users.
type FirebaseUserCreatorAdapter struct {
	client *fbauth.Client
}

func NewFirebaseUserCreator(client *fbauth.Client) *FirebaseUserCreatorAdapter {
	return &FirebaseUserCreatorAdapter{client: client}
}

func (a *FirebaseUserCreatorAdapter) CreateFirebaseUser(ctx context.Context, email, password, displayName string) (string, error) {
	params := (&fbauth.UserToCreate{}).
		Email(email).
		Password(password).
		DisplayName(displayName)

	user, err := a.client.CreateUser(ctx, params)
	if err != nil {
		return "", err
	}
	return user.UID, nil
}

func (a *FirebaseUserCreatorAdapter) DeleteFirebaseUser(ctx context.Context, uid string) error {
	return a.client.DeleteUser(ctx, uid)
}
