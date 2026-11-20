package isolationx

import "github.com/google/uuid"

func sibling() uuid.UUID { return uuid.New() } // want `uuid.New mints a random v4 id`
