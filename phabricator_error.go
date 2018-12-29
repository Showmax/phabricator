package phabricator

// Error denotes an error directly from the API
type Error struct {
	err string
}

func (pe Error) Error() string {
	return pe.err
}
