package shared

import "github.com/google/uuid"

func GenerateUUID() string {
	return uuid.New().String()
}

func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
