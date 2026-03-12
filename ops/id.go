package ops

import "strings"

type Id struct {
	Name    string
	Version *Version
}

func (i *Id) String() string {
	return strings.Join([]string{i.Name, i.Version.String()}, "@")
}

func (i *Id) FromString(str string) error {
	parts := strings.Split(str, "@")
	i.Name = parts[0]
	i.Version = &Version{}

	err := i.Version.Parse(parts[1])
	if err != nil {
		return err
	}

	return nil
}
