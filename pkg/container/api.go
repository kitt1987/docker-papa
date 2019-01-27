package container

type RecreateOptions struct {
	Image            string
	RestartAlways    bool
	Network          string
	Bindings         []string
	RenewBindings    bool
	Env              []string
	RenewEnv         bool
	PortMapping      []string
	RenewPortMapping bool
	Cmd              []string
	RenewCmd         bool
	Rename           string
	KeepFiles        []string
}

type DockerContainer interface {
	Recreate(*RecreateOptions) (newID string, err error)
	ConvertToDockerCommand() (string, error)
}
