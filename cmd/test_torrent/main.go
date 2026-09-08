package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bubble-stream/internal/bittorrent"
	"bubble-stream/internal/sysutil"
)

// Default sample magnet (Friends Season 4 pack)
const defaultSampleMagnet = "magnet:?xt=urn:btih:0ffe2b47322990aa9ce27e3ff9bcdc19884a3f42&dn=Friends+%281994%29+S04+S04+%281080p+BluRay+x265+HEVC+10bit+AAC+5&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce"

func main() {
	// Clean up any temp data on startup and exit
	sysutil.PurgeAllTempData()
	defer sysutil.PurgeAllTempData()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	magnet := defaultSampleMagnet
	if len(os.Args) > 1 {
		magnet = os.Args[1]
	}

	fmt.Println("==================================================")
	fmt.Println("  Goovie Pure Go BitTorrent Test Harness")
	fmt.Println("==================================================")
	fmt.Println("Starting torrent engine (in-process, zero Node.js)...")

	engine, err := bittorrent.NewEngine()
	if err != nil {
		fmt.Printf("Error initializing engine: %v\n", err)
		os.Exit(1)
	}
	defer engine.Close()

	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt. Purging and shutting down...")
		engine.Close()
		sysutil.PurgeAllTempData()
		os.Exit(0)
	}()

	fmt.Printf("\nConnecting to swarm and fetching torrent metadata...\n")
	fmt.Printf("Magnet: %s\n\n", magnet[:min(len(magnet), 70)]+"...")

	startTime := time.Now()
	items, err := engine.FetchFiles(magnet, 30*time.Second)
	if err != nil {
		fmt.Printf("Failed to fetch torrent files: %v\n", err)
		return
	}

	elapsed := time.Since(startTime)
	fmt.Printf("Metadata resolved in %.2f seconds! Found %d files:\n\n", elapsed.Seconds(), len(items))

	for _, item := range items {
		fmt.Printf("  [%02d] %s (%s)\n", item.Index, item.Path, bittorrent.FormatBytes(item.Length))
	}

	fmt.Print("\nEnter file index to stream into MPV (or 'q' to quit): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" || strings.ToLower(input) == "q" {
		fmt.Println("Exiting.")
		return
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 0 || idx >= len(items) {
		fmt.Println("Invalid file index.")
		return
	}

	selected := items[idx]
	fmt.Printf("\nStarting local HTTP stream for file [%d]: %s...\n", selected.Index, selected.Path)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	session, err := engine.StartStreamServer(ctx, magnet, selected.Index)
	if err != nil {
		fmt.Printf("Error starting stream server: %v\n", err)
		return
	}
	defer session.Close()

	fmt.Printf("Local HTTP Stream URL: %s\n", session.StreamURL)
	fmt.Println("Launching MPV player...")

	cmd := exec.Command("mpv", session.StreamURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	fmt.Println("\nPlayback finished. Cleaned up stream session.")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
