package main

type Pattern struct {
	// Root directory
	Path string

	// List of file name patterns
	FileNamePatternList []string

	// List of file path patterns
	FilePathPatternList []string

	// Recursive search
	Recursive bool
}
