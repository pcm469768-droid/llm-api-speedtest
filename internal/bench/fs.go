package bench

import "os"

// 文件读写收在包内,便于测试时替换为临时目录;不引入额外依赖。
var (
	readFile  = os.ReadFile
	writeFile = os.WriteFile
)
