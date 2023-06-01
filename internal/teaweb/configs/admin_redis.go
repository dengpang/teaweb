package configs

type RedisConfig struct {
	Addr     string `yaml:"addr" json:"addr"`
	Password string `yaml:"password" json:"password"`
	DBIndex  int    `yaml:"dbindex" json:"dbindex"`
	PoolSize int    `yaml:"poolSize" json:"poolSize"`
	IdleSize int    `yaml:"idleConns" json:"idleConns"`
}
