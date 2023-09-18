package gmakec

type CompilerFindResult struct {
	Type string
	File string
	Path string
}

type CompilerFindDefinition struct {
	Type    string   `yaml:"type"`
	Names   []string `yaml:"names"`
	Paths   []string `yaml:"paths"`
	Results []CompilerFindResult
}
