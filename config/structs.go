package config

type Config struct {
	Host []struct {
		IP       string `yaml:"ip"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"host"`
	Ingress struct {
		Subnet  string `yaml:"subnet"`
		Gateway string `yaml:"gateway"`
	} `yaml:"ingress"`
	Mysql struct {
		Host   string `yaml:"host"`
		Port   string `yaml:"port"`
		DBName string `yaml:"dbname"`
		User   string `yaml:"user"`
		Passwd string `yaml:"passwd"`
	} `yaml:"mysql"`
}

type ServiceConfig struct {
	Name        string            `json:"Name"`
	Image       string            `json:"Image"`
	Labels      map[string]string `json:"Labels"`
	TargetPort  uint32            `json:"targetPort"`
	PublishPort uint32            `json:"publishPort"`
	Env         []string          `json:"ENV"`
	Host        string            `json:"SvcHost"`
	NodeID      string            `json:"NodeID"`
	RawSvcID    string            `json:"RawSvcID"`
}
