package database

type Driver int

const (
	Postgres Driver = iota
)

// driverMap 注册支持的数据库类型，方便使用
var driverMap = make(map[Driver]string)

// 参考：dialects.regDrvsNDialects
func init() {
	driverMap[Postgres] = "postgres"
}

func (d Driver) String() string {
	return driverMap[d]
}
