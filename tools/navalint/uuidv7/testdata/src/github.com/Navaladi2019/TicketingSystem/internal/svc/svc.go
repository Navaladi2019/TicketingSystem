package svc

import "github.com/google/uuid"

func bad() {
	_ = uuid.New()                  // want `uuid.New mints a random v4 id`
	_ = uuid.NewString()            // want `uuid.NewString mints a random v4 id`
	_, _ = uuid.NewRandom()         // want `uuid.NewRandom mints a random v4 id`
	_ = uuid.Must(uuid.NewRandom()) // want `uuid.NewRandom mints a random v4 id`
	f := uuid.New                   // want `uuid.New mints a random v4 id`
	_ = f
}

func good(ns uuid.UUID) {
	_, _ = uuid.NewV7()
	_ = uuid.NewSHA1(ns, nil)
	_ = uuid.MustParse("x")
}
