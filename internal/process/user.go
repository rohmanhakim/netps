package process

type ProcessUser struct {
	RealUID    int
	Name       string
	Privileged bool
}

func (pu *ProcessUser) PrivilegedString() string {
	if pu.Privileged {
		return "privileged"
	}
	return "unprivileged"
}
