package db

import (
	"fmt"
	"sync"
)

type Container struct {
	ID           string
	FunctionName string
	Status       string
	HostPort     string
}

var (
	ContainerMap = map[string][]*Container{}
	mu           sync.Mutex
)

func PrintContainerMap() {
	mu.Lock()
	defer mu.Unlock()
	for function, containers := range ContainerMap {
		fmt.Println("Function:", function)

		for _, c := range containers {
			fmt.Printf(
				"  ID: %s | Status: %s | Port: %s\n",
				c.ID,
				c.Status,
				c.HostPort,
			)
		}
	}
}

func GetFreeContainer(functionName string) *Container {
	mu.Lock()
	defer mu.Unlock()

	containers := ContainerMap[functionName]
	fmt.Print(containers)

	for _, c := range containers {
		if c.Status == "free" {
			c.Status = "busy"
			return c
		}
	}

	return nil
}

func MarkFree(containerID string) {
	mu.Lock()
	defer mu.Unlock()

	for _, containers := range ContainerMap {
		for _, c := range containers {
			if c.ID == containerID {
				c.Status = "free"
				return
			}
		}
	}
}

func MarkBusy(containerID string) {
	mu.Lock()
	defer mu.Unlock()

	for _, containers := range ContainerMap {
		for _, c := range containers {
			if c.ID == containerID {
				c.Status = "busy"
				return
			}
		}
	}
}

func RemoveContainer(containerID string) {
	mu.Lock()
	defer mu.Unlock()

	for fn, containers := range ContainerMap {

		for i, c := range containers {

			if c.ID == containerID {

				ContainerMap[fn] = append(
					containers[:i],
					containers[i+1:]...,
				)

				return
			}
		}
	}
}

func AddContainer(c *Container) {
	mu.Lock()
	defer mu.Unlock()

	ContainerMap[c.FunctionName] = append(ContainerMap[c.FunctionName], c)
}
