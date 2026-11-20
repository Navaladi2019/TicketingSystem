package tool

import "github.com/google/uuid"

func run() uuid.UUID { return uuid.New() } // want `uuid.New mints a random v4 id`
