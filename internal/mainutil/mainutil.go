package mainutil

import (
	"io"
	"os"
)

func ReadAllStdin() ([]byte, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Mode()&os.ModeCharDevice != 0 {
		return nil, nil
	}
	return io.ReadAll(os.Stdin)
}
