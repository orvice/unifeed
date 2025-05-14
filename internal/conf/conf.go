package conf

var (
	Conf = new(Config)
)

type Config struct {
}

type Mastodon struct {
	Host  string
	Token string
}

type Feed struct {
	Name     string
	Mastodon Mastodon
}

func (c *Config) Print() {

}
