container {
	dependencies = true
	alpine_secdb = false
	secrets      = true
}

binary {
	go_modules   = true
	osv          = true
	oss_index    = true
	nvd          = false
    secrets {
      matchers {
    	  known = ["tfc", "hcp", "tfe", "github", "artifactory", "slack", "aws", "google", "azure"]
      }
    }
}