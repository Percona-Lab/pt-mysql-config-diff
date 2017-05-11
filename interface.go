package main

type configReader interface {
	Entries() map[string]interface{}
	Keys() []string
	Get(string) (interface{}, bool)
	Type() string
}

type config struct {
	configType string
	entries    map[string]interface{}
}

func (c *config) Entries() map[string]interface{} {
	return c.entries
}

func (c *config) Keys() []string {
	keys := []string{}
	for key, _ := range c.entries {
		keys = append(keys, key)
	}
	return keys
}

func (c *config) Get(key string) (interface{}, bool) {
	val, ok := c.entries[key]
	return val, ok
}

func (c *config) Type() string {
	return c.configType
}
