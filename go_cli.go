package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "new" {
		fmt.Println("Usage: goboot new <project-name>")
		os.Exit(1)
	}

	projectName := os.Args[2]
	if err := createProject(projectName); err != nil {
		log.Fatalf("Failed to create project: %v", err)
	}

	fmt.Printf("✅ Project '%s' created successfully!\n", projectName)
}

func createProject(name string) error {
	// 创建项目目录
	if err := os.Mkdir(name, 0755); err != nil {
		return err
	}

	// 创建 go.mod 文件
	goMod := fmt.Sprintf("module %s\n\nrequire (\n\tgithub.com/your-org/goboot latest\n)", name)
	if err := os.WriteFile(filepath.Join(name, "go.mod"), []byte(goMod), 0644); err != nil {
		return err
	}

	// 创建 config.yaml
	configContent := `app:
  name: "` + name + `"

http:
  port: 8080
  addr: 0.0.0.0
  gin_mode: release
`
	if err := os.MkdirAll(filepath.Join(name, "configs"), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(name, "configs", "config.yaml"), []byte(configContent), 0644); err != nil {
		return err
	}

	// 创建 cmd/main.go
	mainGo := `package main

import (
	"flag"
	"goboot/pkg/app"
	"goboot/pkg/config"
	"goboot/pkg/gin"
	"goboot/pkg/logger"
)

var configFile = flag.String("c", "configs/config.yaml", "config file")

func main() {
	flag.Parse()
	cfg := config.NewConfigManager(config.Options{ConfigFile: config.ConfigFile(*configFile)})
	log, _ := logger.NewLogger(cfg)
	httpOpt, _ := gin.NewOption(cfg)
	http, _ := gin.NewServer(log, cfg, httpOpt)
	a, _ := app.New(cfg, http)
	a.Start()
	a.AwaitSignal()
}`
	if err := os.MkdirAll(filepath.Join(name, "cmd"), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(name, "cmd", "main.go"), []byte(mainGo), 0644); err != nil {
		return err
	}

	// 自动执行 go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = name
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
