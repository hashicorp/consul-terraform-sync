container {
	dependencies = true
	alpine_secdb = false
	secrets      = true
}

binary {
	go_modules   = true
	osv          = true
	oss_index    = true
	nvd          = true

	secrets {
		all = true
	}
}
