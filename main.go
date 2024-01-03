package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/spf13/viper"
)

var (
	portStatuses = make(map[string]string)
	mu           sync.RWMutex // Mutex to synchronize access to portStatuses
)

func main() {
	// Viper setup...
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // or viper.SetConfigType("YAML")
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	serverPort := viper.GetString("server.port")
	serverUrl := viper.GetString("server.url")
	headerSecrete := viper.GetString("server.headerSecrete")
	secretCode := viper.GetString("server.secrete")
	go checkPortsContinuously(viper.GetStringMapString("ports")) // Start checking ports in the background
	app := fiber.New()
	app.Use(logger.New())

	// Apply the secret code middleware to the route
	app.Get("/"+serverUrl, checkSecret(headerSecrete, secretCode), func(c *fiber.Ctx) error {
		mu.RLock()
		defer mu.RUnlock()
		return c.JSON(fiber.Map{
			"status": "running",
			"ports":  portStatuses,
		})
	})

	app.Listen(":" + serverPort)

}

// checkSecret is a middleware to validate the secret code
func checkSecret(headerSecrete string, secretCode string) fiber.Handler {

	return func(c *fiber.Ctx) error {
		if c.Get(headerSecrete) != secretCode {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}
		return c.Next()
	}
}

func checkPortsContinuously(portMap map[string]string) {
	for {
		mu.Lock()
		for port, name := range portMap {
			isOpen := checkPort(port)
			portStatuses[name] = isOpen
		}
		mu.Unlock()

		printStatuses() // Print the current statuses to the terminal

		time.Sleep(5 * time.Second) // Check every 5 seconds
	}
}

func printStatuses() {
	mu.RLock()
	defer mu.RUnlock()

	green := "\033[32m" // ANSI color code for green
	red := "\033[31m"   // ANSI color code for red
	reset := "\033[0m"  // ANSI code to reset color

	for name, status := range portStatuses {
		color := green
		if status == "Closed" {
			color = red
		}
		fmt.Printf("Port %s: %s%s%s\n", name, color, status, reset)
	}
	fmt.Println("Clock: ", time.Now().Format("15:04:05"))
	fmt.Println("----") // Separator for each update
}

func checkPort(port string) string {
	_, err := net.DialTimeout("tcp", "localhost:"+port, time.Second)
	if err != nil {
		return "Closed"
	}
	return "Open"
}
