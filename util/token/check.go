package token

import (
	"errors"
	"strings"
)

func parseBearerToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", errors.New("unauthorized: token is missing")
	}

	if !strings.HasPrefix(tokenString, "Bearer ") || len(tokenString) <= len("Bearer ") {
		return "", errors.New("unauthorized: invalid token format")
	}

	return strings.TrimSpace(tokenString[len("Bearer "):]), nil
}

func CheckToken(tokenString string) (int64, string, error) {
	tokenStr, err := parseBearerToken(tokenString)
	if err != nil {
		return 0, "", err
	}

	// Validate token
	userInterface, err := ParseAccessToken(tokenStr)
	if err != nil {
		err = errors.New("unauthorized: " + err.Error())
		return 0, "", err
	}

	return userInterface.ID, userInterface.Role, nil
}

func CheckRefreshToken(tokenString string) (int64, string, error) {
	tokenStr, err := parseBearerToken(tokenString)
	if err != nil {
		return 0, "", err
	}

	// Validate token
	userInterface, err := ParseRefreshToken(tokenStr)
	if err != nil {
		err = errors.New("unauthorized: " + err.Error())
		return 0, "", err
	}

	return userInterface.ID, userInterface.Role, nil
}
