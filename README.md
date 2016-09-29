#log
go语言的日志库

#安装
```shell
go get github.com/nybuxtsui/log
```

#快速使用
```golang
package main

import (
	"github.com/nybuxtsui/log"
)

func main() {
	log.Init(nil)
	log.Debug("hello")
}
```
输出结果:
```
D1001 200102 a.go:9] hello
```

#配置
配置文件使用toml格式
例子：
```
[[logger]]
name = "default"
writer = "console"
level = "debug"

[[logger]]
name = "default"
writer = "file:mylog"
level = "debug"

[[logger]]
name = "hour"
writer = "file_hour:mylog"
level = "debug"
```
说明：
此处定义了2个日志对象，名称都为```default```。```default```为默认日志对象。log包中的全局函数```Debug()```/```Info()```/```Error()```/...使用```default```日志对象。在这个配置中，定义了2个```default```日志对象，则表示对于输出到```default```日志的会同时输出到这2个日志对象上。第一个日志对象的输出设备为```console```表示为控制台。第二个日志对象的输出设备为```file```，表示输出到文件。```file```设备的格式为```file:<文件前缀>```。level代表的是该日志对象的输出最小级别。
将该文件保存为：```logger.conf```(文件可以随意取名）
修改源代码：
```golang
package main

import (
	"github.com/nybuxtsui/log"
)

func main() {
	log.InitFromFile("logger.conf")
	log.Debug("hello")
}
```
再次运行程序，可以看到除了在屏幕上打印日志，同时还生成了一个`mylog-141001.log`的文件。

