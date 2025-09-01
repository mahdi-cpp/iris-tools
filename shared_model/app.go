package shared_model

type AppConfig struct {
	Version  string   `json:"version"`
	Features []string `json:"features"`
	Enabled  bool     `json:"enabled"`
	Version2 string   `json:"version2"`
}

type AppSetting struct {
	Name  string `json:"name"`
	Logs  int    `json:"logs"`
	Email string `json:"email"`
}
