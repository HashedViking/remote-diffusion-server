package utils

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

func IsValidUUID(key string) bool {
	_, err := uuid.Parse(key)
	return err == nil
}

func GenerateUserKey() string {
	key, err := uuid.NewRandom()
	if err != nil {
		fmt.Println("Error generating random UUID:", err)
		key = uuid.New()
	}
	// uuidStr := strings.ReplaceAll(id.String(), "-", "")
	return key.String()
}

func ExecutionTime(f func()) {
	start := time.Now()
	f()
	fmt.Println("Execution time:", time.Since(start))
}
