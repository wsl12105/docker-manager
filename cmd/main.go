package main

import (
	"fmt"
	"os"

	"github.com/wsl12105/docker-manager/internal/docker"
	"github.com/wsl12105/docker-manager/internal/ui"
)

func main() {
	
	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Printf("❌ Unable to create Docker client: %v\n", err)
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
		fmt.Printf("❌%v\n", err)
		fmt.Println("Please start the Docker service:")
		fmt.Println("  • Linux: sudo systemctl start docker")
		//fmt.Println("  • macOS: open -a Docker")
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
