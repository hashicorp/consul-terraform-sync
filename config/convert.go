package config

import "time"

// Bool returns a pointer to the given bool.
func Bool(b bool) *bool {
	return &b
}

// BoolVal returns the value of the boolean at the pointer, or false if the
// pointer is nil.
func BoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// BoolCopy returns a copy of the boolean pointer
func BoolCopy(b *bool) *bool {
	if b == nil {
		return nil
	}

	return Bool(*b)
}

// BoolPresent returns a boolean indicating if the pointer is nil, or if the
// pointer is pointing to the zero value..
func BoolPresent(b *bool) bool {
	if b == nil {
		return false
	}
	return true
}

// Int returns a pointer to the given int.
func Int(i int) *int {
	return &i
}

// IntVal returns the value of the int at the pointer, or 0 if the pointer is
// nil.
func IntVal(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// String returns a pointer to the given string.
func String(s string) *string {
	return &s
}

// StringVal returns the value of the string at the pointer, or "" if the
// pointer is nil.
func StringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// StringCopy returns a copy of the string pointer
func StringCopy(s *string) *string {
	if s == nil {
		return nil
	}

	return String(*s)
}

// StringPresent returns a boolean indicating if the pointer is nil, or if the
// pointer is pointing to the zero value.
func StringPresent(s *string) bool {
	if s == nil {
		return false
	}
	return *s != ""
}

// TimeDuration returns a pointer to the given time.Duration.
func TimeDuration(t time.Duration) *time.Duration {
	return &t
}

// TimeDurationVal returns the value of the string at the pointer, or 0 if the
// pointer is nil.
func TimeDurationVal(t *time.Duration) time.Duration {
	if t == nil {
		return time.Duration(0)
	}
	return *t
}

// TimeDurationCopy returns a copy of the time.Duration pointer
func TimeDurationCopy(t *time.Duration) *time.Duration {
	if t == nil {
		return nil
	}

	return TimeDuration(*t)
}

// TimeDurationPresent returns a boolean indicating if the pointer is nil, or
// if the pointer is pointing to the zero value.
func TimeDurationPresent(t *time.Duration) bool {
	if t == nil {
		return false
	}
	return *t != 0
}
