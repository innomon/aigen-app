package descriptors

type TemporaryAccessConfig struct {
	Path string `yaml:"path" json:"path"`
	TTL  int    `yaml:"ttl_sec" json:"ttl_sec"`
	Role string `yaml:"role" json:"role"`
}
