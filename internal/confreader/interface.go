package confreader

type ConfigReader interface {
	Get(string) (interface{}, bool)
	Keys() []string
	Entries() map[string]interface{}
	Type() string
}

type Config struct {
	ConfigType string
	EntriesMap map[string]interface{}
}

func (c *Config) Keys() []string {
	keys := []string{}
	for key, _ := range c.EntriesMap {
		keys = append(keys, key)
	}
	return keys
}

func (c *Config) Get(key string) (interface{}, bool) {
	val, ok := c.EntriesMap[key]
	return val, ok
}

func (c *Config) Entries() map[string]interface{} {
	return c.EntriesMap
}

func (c *Config) Type() string {
	return c.ConfigType
}
