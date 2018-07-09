package storage

// TestSuite is common interface which each storage needs to implement
type TestSuite interface {
	TestNotFound()
}
