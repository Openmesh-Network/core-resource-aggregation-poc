package p2p

// Validator is a validator that always returns valid
type Validator struct {
}

// Validate is to determine whether a key is valid
func (v *Validator) Validate(key string, value []byte) error {
    // nil = valid
    return nil
}

// Select returns the index of the best value and nil, or -1 and an error if none are valid
func (v *Validator) Select(key string, values [][]byte) (int, error) {
    return 0, nil
}
