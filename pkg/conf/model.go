package conf

type AccountConfig struct {
	BiliBili BiliBiliConfig `mapstructure:"bilibili" json:"bilibili" yaml:"bilibili"`
	Huya     HuyaConfig     `mapstructure:"huya" json:"huya" yaml:"huya"`
	Douyu    DouyuConfig    `mapstructure:"douyu" json:"douyu" yaml:"douyu"`
}

type HuyaConfig struct {
	Enable  bool   `mapstructure:"enable" json:"enable" yaml:"enable"`
	Cookies string `mapstructure:"cookies" json:"cookies,omitempty" yaml:"cookies"`
}

type DouyuConfig struct {
	Enable bool `mapstructure:"enable" json:"enable" yaml:"enable"`
}

type BiliBiliConfig struct {
	Enable          bool   `mapstructure:"enable" json:"enable" yaml:"enable"`
	DedeUserID      string `mapstructure:"DedeUserID" json:"DedeUserID,omitempty" yaml:"DedeUserID"`                // DedeUserID
	DedeUserIDCkMd5 string `mapstructure:"DedeUserIDCkMd5" json:"DedeUserIDCkMd5,omitempty" yaml:"DedeUserIDCkMd5"` // DedeUserID__ckMd5
	SESSDATA        string `mapstructure:"SESSDATA" json:"SESSDATA,omitempty" yaml:"SESSDATA"`                      // SESSDATA
	BiliJCT         string `mapstructure:"BiliJCT" json:"BiliJCT,omitempty" yaml:"BiliJCT"`                         // bili_jct
}
