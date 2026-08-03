package testsupport

type HostBaseline struct {
	Processes  int `json:"processes"`
	Interfaces int `json:"interfaces"`
	Namespaces int `json:"namespaces"`
	Rules      int `json:"rules"`
	Artifacts  int `json:"artifacts"`
}

func (b HostBaseline) Restored(final HostBaseline) bool { return b == final }
