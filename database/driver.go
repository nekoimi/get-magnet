package database

type Driver int

const (
	Postgres Driver = iota
	MySQL
)

// driverMap 注册支持的数据库类型，方便使用
var driverMap = make(map[Driver]string)

// 参考：dialects.regDrvsNDialects
func init() {
	driverMap[Postgres] = "postgres"
	driverMap[MySQL] = "mysql"
}

func (d Driver) String() string {
	return driverMap[d]
}