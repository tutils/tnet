package tcp

import (
	"log"
	"sync/atomic"
	"time"
)

var (
	DefaultErrorLogFunc = log.Printf
)

type atomicBool int32

func (b *atomicBool) isSet() bool { return atomic.LoadInt32((*int32)(b)) != 0 }
func (b *atomicBool) setTrue()    { atomic.StoreInt32((*int32)(b), 1) }

var shutdownPollInterval = 500 * time.Millisecond
