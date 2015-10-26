package utils

import (
	"math/rand"
	"time"

	"github.com/satori/go.uuid"
)

func randRange(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

// UUID returns a id
func UUID() uuid.UUID {
	return uuid.NewV4()
}

// RandomDuration returns a random duration between 150 and 350 milliseconds
func RandomDuration() time.Duration {
	return time.Duration(randRange(3000, 5000)) * time.Millisecond
}
