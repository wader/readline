package readline

func correctErrNo0(e error) error {
	// errno 0 means everything is ok :)
	if e == nil || e.Error() == "errno 0" {
		return nil
	}
	return e
}
