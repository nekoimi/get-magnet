package bus

type Type int

const (
	Download Type = iota
	SubmitTask
	SubmitJavDB
	Aria2Test
	Aria2LinkUp
	Aria2LinkDown
	Aria2Refresh
)

var (
	eventBus *Bus
	eventMap = make(map[Type]string)
)

func init() {
	eventBus = newEventBus()
	// 下载事件
	eventMap[Download] = "event.download"
	// 提交任务
	eventMap[SubmitTask] = "event.submit.task"
	eventMap[SubmitJavDB] = "event.submit.javdb"
	// arta2测试
	eventMap[Aria2Test] = "event.aria2.test"
	// aria2上线
	eventMap[Aria2LinkUp] = "event.aria2.link_up"
	// aria2下线
	eventMap[Aria2LinkDown] = "event.aria2.link_down"
	// aria2连接配置更新
	eventMap[Aria2Refresh] = "event.aria2.refresh"
}

func Event() *Bus {
	return eventBus
}

func (t Type) Topic() string {
	return eventMap[t]
}
