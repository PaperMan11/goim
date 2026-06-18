package internal

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
