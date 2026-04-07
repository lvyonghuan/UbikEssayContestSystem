package token

import (
	"errors"
)

func CheckToken(tokenString string) (int64, string, error) {
	// Get token from header
	if tokenString == "" {
		err := errors.New("unauthorized: token is missing")
		return 0, "", err
	}

	tokenStr := tokenString[len("Bearer "):]

	// Validate token
	userInterface, err := ParseAccessToken(tokenStr)
	if err != nil {
		err = errors.New("unauthorized: " + err.Error())
		return 0, "", err
	}

	return userInterface.ID, userInterface.Role, nil
}

func CheckRefreshToken(tokenString string) (int64, string, error) {
	// Get token from header
	if tokenString == "" {
		err := errors.New("unauthorized: token is missing")
		return 0, "", err
	}

	tokenStr := tokenString[len("Bearer "):]

	// Validate token
	userInterface, err := ParseRefreshToken(tokenStr)
	if err != nil {
		err = errors.New("unauthorized: " + err.Error())
		return 0, "", err
	}

	return userInterface.ID, userInterface.Role, nil
}
