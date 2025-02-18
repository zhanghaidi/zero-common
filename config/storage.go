package config

var GlobalStorage StorageConf

type StorageConf struct {
	Driver string `json:",default=local,options=[local,oss]"` // local/oss
	Local  struct {
		Directory string `json:",default=storage"`
		BaseUrl   string `json:",optional"`
	} `json:",optional"`
	Oss struct {
		Endpoint        string `json:",optional"`
		AccessKeyID     string `json:",optional"`
		AccessKeySecret string `json:",optional"`
		BucketName      string `json:",optional"`
		BucketURL       string `json:",optional"`
	} `json:",optional"`
}
