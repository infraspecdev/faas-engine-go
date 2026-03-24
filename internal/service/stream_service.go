package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"faas-engine-go/internal/sdk"
	"faas-engine-go/internal/sqlite/models"
)

type LogStreamService struct {
	containerClient sdk.ContainerClient
	store           Store
}

func NewLogStreamService(c sdk.ContainerClient, s Store) *LogStreamService {
	return &LogStreamService{
		containerClient: c,
		store:           s,
	}
}

func (s *LogStreamService) StreamFunctionLogs(
	ctx context.Context,
	functionID int,
	out chan<- string,
) error {

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	activeStreams := make(map[string]bool)
	var mu sync.Mutex

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			containers, err := s.store.GetContainersByFunction(functionID)
			if err != nil {
				out <- fmt.Sprintf("error: %v", err)
				continue
			}

			running := s.filterRunningContainers(ctx, containers)
			if len(running) == 0 {
				continue
			}

			for _, c := range running {
				mu.Lock()
				if activeStreams[c.ID] {
					mu.Unlock()
					continue
				}
				activeStreams[c.ID] = true
				mu.Unlock()

				go func(containerID string) {
					s.streamSingleContainer(ctx, containerID, out)

					mu.Lock()
					delete(activeStreams, containerID)
					mu.Unlock()
				}(c.ID)
			}
		}
	}
}

func (s *LogStreamService) filterRunningContainers(
	ctx context.Context,
	containers []models.Container,
) []models.Container {

	var wg sync.WaitGroup
	resultChan := make(chan models.Container, len(containers))
	sem := make(chan struct{}, 10)

	for _, c := range containers {
		wg.Add(1)

		go func(container models.Container) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			inspect, err := s.containerClient.InspectContainer(ctx, container.ID)
			if err != nil {
				return
			}

			if inspect.Container.State != nil && inspect.Container.State.Running {
				resultChan <- container
			}
		}(c)
	}

	wg.Wait()
	close(resultChan)

	var result []models.Container
	for c := range resultChan {
		result = append(result, c)
	}

	return result
}

func (s *LogStreamService) streamSingleContainer(
	ctx context.Context,
	containerID string,
	out chan<- string,
) {

	reader, err := s.containerClient.StreamContainerLogs(ctx, containerID)
	if err != nil {
		slog.Error("stream logs failed", "container", containerID, "error", err)
		return
	}
	defer reader.Close()

	header := make([]byte, 8)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, err := io.ReadFull(reader, header)
		if err != nil {

			if err == io.EOF || strings.Contains(err.Error(), "context canceled") {
				return
			}

			slog.Error("header read error", "container", containerID, "error", err)
			return
		}

		length := binary.BigEndian.Uint32(header[4:8])
		if length == 0 {
			continue
		}

		payload := make([]byte, length)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			slog.Error("payload read error", "container", containerID, "error", err)
			return
		}

		line := strings.TrimSpace(string(payload))
		if line == "" {
			continue
		}

		out <- fmt.Sprintf("[invoke %s] %s", containerID[:6], line)
	}
}

func (s *LogStreamService) GetFunctionID(name string) (int, error) {
	fn, err := s.store.GetActiveFunction(name)
	if err != nil {
		return 0, err
	}

	if fn == nil {
		return 0, fmt.Errorf("function not found")
	}

	return fn.ID, nil
}
