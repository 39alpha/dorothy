package core

func Clone(remote string) error {
	if err := Init(); err != nil {
		return err
	}

	if err := SetRemote(remote); err != nil {
		return err
	}

	if err := Fetch(); err != nil {
		return err
	}

	return nil
}
