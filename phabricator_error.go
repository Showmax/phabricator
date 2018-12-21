package phabricator

type Error struct {
	err string
}

func (pe Error) Error() string {
	return pe.err
}
