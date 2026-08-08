package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	var protoFiles []string

	// Quét toàn bộ file .proto trong thư mục proto/
	err := filepath.Walk("proto", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".proto" {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("❌ Lỗi khi tìm kiếm thư mục proto: %v\n", err)
		os.Exit(1)
	}

	if len(protoFiles) == 0 {
		fmt.Println("⚠️ Không tìm thấy file .proto nào trong thư mục proto/")
		os.Exit(1)
	}

	fmt.Printf("🔍 Đã tìm thấy %d file proto. Bắt đầu biên dịch...\n", len(protoFiles))
	for _, f := range protoFiles {
		fmt.Printf("   📄 %s\n", f)
	}

	// Cấu hình protoc: Đẩy kết quả sinh ra vào pkg/genproto/ dựa theo module name 'paynexus'
	args := []string{
		"--proto_path=.",
		"--go_out=.",
		"--go_opt=module=paynexus",
		"--go-grpc_out=.",
		"--go-grpc_opt=module=paynexus",
	}
	args = append(args, protoFiles...)

	cmd := exec.Command("protoc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("\n❌ Lỗi khi thực thi protoc: %v\n", err)
		fmt.Println("💡 Gợi ý: Hãy đảm bảo 'protoc', 'protoc-gen-go', và 'protoc-gen-go-grpc' đã được cài đặt và thêm vào PATH của hệ thống.")
		os.Exit(1)
	}

	fmt.Println("\n✅ Biên dịch Protobuf thành công! Code Go đã được đẩy vào thư mục 'pkg/genproto/'.")
}
