package goczmq

import "testing"

func TestPoller(t *testing.T) {
	pullSock1, err := NewPull("inproc://poller_pull1")
	if err != nil {
		t.Errorf("NewPull failed: %s", err)
	}
	defer pullSock1.Destroy()

	poller, err := NewPoller(pullSock1)
	if err != nil {
		t.Errorf("NewPoller failed: %s", err)
	}
	defer poller.Destroy()

	if len(poller.socks) != 1 {
		t.Errorf("Expected number of socks to be 1, was %d", len(poller.socks))
	}

	pullSock2, err := NewPull("inproc://poller_pull2")
	if err != nil {
		t.Errorf("NewPull failed: %s", err)
	}
	defer pullSock2.Destroy()

	err = poller.Add(pullSock2)
	if err != nil {
		t.Errorf("poller Add failed: %s", err)
	}

	if len(poller.socks) != 2 {
		t.Errorf("Expected number of socks to be 2, was %d", len(poller.socks))
	}

	poller.Destroy()
	poller, err = NewPoller(pullSock1, pullSock2)
	if err != nil {
		t.Errorf("NewPoller failed: %s", err)
	}

	if len(poller.socks) != 2 {
		t.Errorf("Expected number of zsocks to be 2, was %d", len(poller.socks))
	}

	if poller.socks[0].zsockT != pullSock1.zsockT || poller.socks[1].zsockT != pullSock2.zsockT {
		t.Error("Expected each passed zsock to be in the poller")
	}

	pushSock, err := NewPush("inproc://poller_pull1")
	if err != nil {
		t.Errorf("NewPush failed: %s", err)
	}
	defer pushSock.Destroy()

	err = pushSock.SendFrame([]byte("Hello"), FlagNone)
	if err != nil {
		t.Errorf("SendMessage failed: %s", err)
	}

	s := poller.Wait(0)
	if s == nil {
		t.Errorf("Wait did not return waiting socket")
	}

	frame, _, err := s.RecvFrame()
	if err != nil {
		t.Errorf("RecvMessage failed: %s", err)
	}

	if string(frame) != "Hello" {
		t.Errorf("Expected 'Hello', received %s", string(frame))
	}

	pushSock2, err := NewPush("inproc://poller_pull2")
	if err != nil {
		t.Errorf("NewPush failed: %s", err)
	}

	err = pushSock2.SendFrame([]byte("World"), FlagNone)
	if err != nil {
		t.Errorf("SendMessage failed: %s", err)
	}

	s = poller.Wait(0)
	if s == nil {
		t.Errorf("Wait did not return waiting socket")
	}

	frame, _, err = s.RecvFrame()
	if err != nil {
		t.Errorf("RecvMessage failed: %s", err)
	}

	if string(frame) != "World" {
		t.Errorf("Expected 'World', received %s", string(frame))
	}

	poller.Remove(pullSock2)
	if len(poller.socks) != 1 {
		t.Errorf("socks len should be 1 after removing pushsock, is %d", len(poller.socks))
	}
}

func ExamplePoller() {
	sock1, err := NewRouter("inproc://poller_example_1")
	if err != nil {
		panic(err)
	}
	defer sock1.Destroy()

	poller, err := NewPoller(sock1)
	if err != nil {
		panic(err)
	}

	sock2, err := NewRouter("inproc://poller_example_2")
	if err != nil {
		panic(err)
	}
	defer sock2.Destroy()

	err = poller.Add(sock2)
	if err != nil {
		panic(err)
	}

	// Poller.Wait(millis) returns first socket that has a waiting message
	_ = poller.Wait(1)
}

func benchmarkPollerSendFrame(size int, b *testing.B) {
	pullSock := NewSock(Pull)
	defer pullSock.Destroy()

	_, err := pullSock.Bind("inproc://benchSock")
	if err != nil {
		panic(err)
	}

	go func() {
		pushSock := NewSock(Push)
		defer pushSock.Destroy()
		err := pushSock.Connect("inproc://benchSock")
		if err != nil {
			panic(err)
		}

		payload := make([]byte, size)
		for i := 0; i < b.N; i++ {
			err = pushSock.SendFrame(payload, FlagNone)
			if err != nil {
				panic(err)
			}
		}
	}()

	poller, err := NewPoller(pullSock)
	if err != nil {
		panic(err)
	}
	defer poller.Destroy()

	for i := 0; i < b.N; i++ {
		s := poller.Wait(-1)
		msg, _, err := s.RecvFrame()
		if err != nil {
			panic(err)
		}
		if len(msg) != size {
			panic("msg too small")
		}
	}
}

func BenchmarkPollerSendFrame1k(b *testing.B)  { benchmarkPollerSendFrame(1024, b) }
func BenchmarkPollerSendFrame4k(b *testing.B)  { benchmarkPollerSendFrame(4096, b) }
func BenchmarkPollerSendFrame16k(b *testing.B) { benchmarkPollerSendFrame(16384, b) }
func BenchmarkPollerSendFrame65k(b *testing.B) { benchmarkPollerSendFrame(65536, b) }
