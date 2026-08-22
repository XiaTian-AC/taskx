package bg

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"
)

func Watch(logPath, id string, isAlive func() bool, out io.Writer) error {
	f, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("no log for %s: %w", id, err)
	}
	defer f.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)

	fmt.Fprintf(out, "--- tkx watch %s (Ctrl+C to exit, task keeps running) ---\n", id)

	var offset int64
	drain(f, out, &offset)

	for {
		select {
		case <-interrupt:
			fmt.Fprintf(out, "\n--- exiting watch (task %s still running) ---\n", id)
			return nil
		default:
		}

		if st, err := os.Stat(logPath); err == nil {
			if st.Size() < offset {
				f.Seek(0, io.SeekStart)
				offset = 0
			}
			if st.Size() > offset {
				drain(f, out, &offset)
			}
		}

		if isAlive != nil && !isAlive() {
			if st, err := os.Stat(logPath); err == nil && st.Size() > offset {
				drain(f, out, &offset)
			}
			fmt.Fprintf(out, "--- %s has exited ---\n", id)
			return nil
		}

		time.Sleep(200 * time.Millisecond)
	}
}

func drain(f *os.File, out io.Writer, offset *int64) {
	buf := make([]byte, 32*1024)
	for {
		n, err := f.ReadAt(buf, *offset)
		if n > 0 {
			out.Write(buf[:n])
			*offset += int64(n)
		}
		if err != nil || n == 0 {
			return
		}
	}
}
