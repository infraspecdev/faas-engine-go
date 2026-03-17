package db

import (
	"fmt"
	"sync"
	"time"
)

type Container struct {
	ID           string
	FunctionName string
	Status       string
	HostPort     string
	LastUsed     time.Time
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

func ResetContainerMap() {
	mu.Lock()
	defer mu.Unlock()

	ContainerMap = map[string][]*Container{}
}

func GetAllContainers() []*Container {
	mu.Lock()
	defer mu.Unlock()

	var result []*Container

	for _, containers := range ContainerMap {
		result = append(result, containers...)
	}

	return result
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
				c.LastUsed = time.Now()
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

	c.LastUsed = time.Now()

	ContainerMap[c.FunctionName] = append(ContainerMap[c.FunctionName], c)
}

func CleanupIdleContainers(timeout time.Duration, cleanup func(string)) {
	mu.Lock()

	var toCleanup []string

	for fn, containers := range ContainerMap {
		var active []*Container

		for _, c := range containers {
			if c.Status == "free" && time.Since(c.LastUsed) > timeout {
				toCleanup = append(toCleanup, c.ID)
				continue
			}
			active = append(active, c)
		}

		ContainerMap[fn] = active
	}

	mu.Unlock()

	// Run cleanup AFTER releasing lock
	for _, id := range toCleanup {
		go cleanup(id)
	}
}
