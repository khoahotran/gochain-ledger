package main

import (
	"os"

	"github.com/khoahotran/gochain-ledger/cmd"
)

func main() {
	// Khi chạy, luôn dọn dẹp (đóng DB) khi thoát
	defer func() {
		// (Chúng ta sẽ thêm logic đóng DB ở đây sau)
	}()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
