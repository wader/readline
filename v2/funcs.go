package readline

import (
	"sync"
	"time"
)

// WaitForResume need to call before current process got suspend.
// It will run a ticker until a long duration is occurs,
// which means this process is resumed.
func WaitForResume() chan struct{} {
	ch := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		t := time.Now()
		wg.Done()
		for {
			now := <-ticker.C
			if now.Sub(t) > 100*time.Millisecond {
				break
			}
			t = now
		}
		ticker.Stop()
		ch <- struct{}{}
	}()
	wg.Wait()
	return ch
}
