package loginstrategy

type LoginStrategy string

const (
	LoginStrategyAllowMulti          LoginStrategy = "allow_multi"
	LoginStrategySingle              LoginStrategy = "single"
	LoginStrategyReplace             LoginStrategy = "replace"
	LoginStrategyReplaceSamePlatform LoginStrategy = "replace_same_platform"
)

func (s LoginStrategy) Validate() bool {
	switch s {
	case LoginStrategyAllowMulti, LoginStrategySingle, LoginStrategyReplace, LoginStrategyReplaceSamePlatform:
		return true
	default:
		return false
	}
}

type LoginStrategyConf struct {
	LoginStrategy             LoginStrategy `json:",default=allow_multi"` // 多端登录策略
	MaxConnPerUser            int64         `json:",default=10"`          // 每个用户最大连接数（allow_multi策略下生效）
	MaxConnPerUserPerPlatform int64         `json:",default=3"`           // 每个用户每个平台最大连接数
}
