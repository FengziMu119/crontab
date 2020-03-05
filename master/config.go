package master

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	ApiPort               int      `json:"apiPort"`
	ApiReadTimeout        int      `json:"apiReadTimeout"`
	AipWriteTimeout       int      `json:"aipWriteTimeout"`
	EtcdEndpoints         []string `json:"etcdEndpoints"`
	EtcdDialTimeout       int      `json:"etcdDialTimeout"`
	WebRoot               string   `json:"webRoot"`
	MongoUri              string   `json:"mongoUri"`
	MongodbConnectTimeOut int      `json:"mongodbConnectTimeOut"`
}

// 配置单例
var (
	G_config *Config
)

// 加载配置
func InitConfig(fileName string) (err error) {
	var (
		content []byte
		conf    Config
	)
	// 1 读取配置文件
	if content, err = ioutil.ReadFile(fileName); err != nil {
		return
	}
	// 2 反序列化Json
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	// 3 赋值单例
	G_config = &conf

	return
}
