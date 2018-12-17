package phabricator

type PhabricatorError struct {
	err string
}

func (pe PhabricatorError) Error() string {
	return pe.err
}
