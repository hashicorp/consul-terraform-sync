package config

func mergeSlices(orig, incoming []string) []string {
	if orig == nil {
		if incoming == nil {
			return nil
		}
		return incoming
	}

	if incoming == nil {
		return orig
	}

	for _, incomingVal := range incoming {
		// only add an incoming value if it does not already exist in the
		// original slice
		exists := false
		for _, origVal := range orig {
			if incomingVal == origVal {
				exists = true
				break
			}
		}

		if !exists {
			orig = append(orig, incomingVal)
		}
	}

	return orig
}
