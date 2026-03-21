package config

// Default user configuration — override via ~/.jterrazz/config.json
const (
	defaultUserEmail = "admin@jterrazz.com"
	defaultUserName  = "Jean-Baptiste Terrazzoni"
)

// UserEmail returns the configured email (from config.json or default)
func UserEmail() string {
	cfg, err := LoadJRC()
	if err == nil && cfg.UserEmail != "" {
		return cfg.UserEmail
	}
	return defaultUserEmail
}

// UserName returns the configured name (from config.json or default)
func UserName() string {
	cfg, err := LoadJRC()
	if err == nil && cfg.UserName != "" {
		return cfg.UserName
	}
	return defaultUserName
}
