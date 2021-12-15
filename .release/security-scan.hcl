container {
	dependencies = true
	alpine_secdb = true
	secrets      = false
}

binary {
	secrets      = false
	go_modules   = true
	osv          = true
	oss_index    = true
	nvd          = true
}
