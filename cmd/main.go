package main

import (
	"fmt"
	"os"

	"github.com/wsl12105/docker-manager/internal/docker"
	"github.com/wsl12105/docker-manager/internal/ui"
)

const (
    
    styleBold      = "\033[1m"
    styleUnderline = "\033[4m"
    styleItalic    = "\033[3m"
    styleReset     = "\033[0m"
    
    
    colorRed    = "\033[31m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
    colorBlue   = "\033[34m"
    colorPurple = "\033[35m"
    colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
)

func main() {
	
	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Printf(colorRed + "❌ Unable to create Docker client: %v\n" + colorReset, err)
		fmt.Println("Please ensure that Docker is installed and configured correctly")
		fmt.Println("\nPress Ctrl+C to Exit")
		
	
		done := make(chan bool)
		go func() {
			os.Stdin.Read(make([]byte, 1))
			done <- true
		}()
		<-done
		os.Exit(1)
	}
	defer dockerClient.Close()

	
	if err := dockerClient.CheckDockerRunning(); err != nil {
		fmt.Printf(colorRed + styleBold + "❌ %v\n" + colorReset, err)
		fmt.Println("Please start the Docker service:")
		fmt.Println(colorGreen + "  • Linux: sudo systemctl start docker" + colorReset)
		fmt.Println("  • macOS: open -a Docker")
		fmt.Println("\nPress Ctrl+C to Exit")
		

		done := make(chan bool)
		go func() {
			os.Stdin.Read(make([]byte, 1))
			done <- true
		}()
		<-done
		os.Exit(1)
	}


	app := ui.NewApp(dockerClient)

	if err := app.Run(); err != nil {
		fmt.Printf("❌ DM execution failed: %v\n", err)
		os.Exit(1)
	}
}
