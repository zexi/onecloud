package apis

type ContainerKeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ContainerSpec struct {
	// Image to use.
	Image string `json:"image"`
	// Command to execute (i.e., entrypoint for docker)
	Command []string `json:"command"`
	// Args for the Command (i.e. command for docker)
	Args []string `json:"args"`
	// Current working directory of the command.
	WorkingDir string `json:"working_dir"`
	// List of environment variable to set in the container.
	Envs []*ContainerKeyValue `json:"envs"`
}
