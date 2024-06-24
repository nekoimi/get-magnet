package test

import "testing"

func TestDead(t *testing.T) {
	ch := make(chan int)
	ch <- 1 // 这里会导致死锁，因为没有接收者
}

func TestDead2(t *testing.T) {
	ch := make(chan int)
	<-ch // 这里会导致死锁，因为没有发送者
}

func TestDeadOK(t *testing.T) {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
	<-ch
}

func TestDead3(t *testing.T) {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
	// 主协程没有等待接收就退出了
}
