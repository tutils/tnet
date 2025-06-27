package main

import "C"
import (
	"log"
	"os"
	"unsafe"

	"github.com/tutils/tnet/cmd"
)

//export RunCmd
func RunCmd(cargs **C.char, size C.int) {
	log.SetFlags(log.Ltime | log.Lshortfile)

	// 将 C 字符串数组转换为 Go []string
	args := os.Args[:1]
	ptr := unsafe.Pointer(cargs)
	for i := 0; i < int(size); i++ {
		// 获取第 i 个元素的指针
		cStrPtr := (**C.char)(unsafe.Pointer(uintptr(ptr) + uintptr(i)*unsafe.Sizeof(uintptr(0))))
		args = append(args, C.GoString(*cStrPtr))
	}
	os.Args = args
	cmd.Execute()
}

func main() {} // 必须的空白主函数
