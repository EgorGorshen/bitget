package config

var Config *ConfigType = new(ConfigType)

type ConfigType struct {
	ApiKey            string
	SecretKey         string
	PASSPHRASE        string
	SignType          string
	TimeoutMilisecond int
}

func (c *ConfigType) Set(ApiKey, SecretKey, PASSPHRASE, SignType string, TimeoutSecond int) {
	c.ApiKey = ApiKey
	c.SecretKey = SecretKey
	c.PASSPHRASE = PASSPHRASE
	c.SignType = SignType
	c.TimeoutMilisecond = TimeoutSecond
}
