package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

func printError(err error) {
	fmt.Println("\n/!\\    Error    /!\\")
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("- - - - - - - - - -")
	fmt.Println(err)
}

func exitWithError(err error) {
	fmt.Println("\n/!\\ Fatal Error /!\\")
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("- - - - - - - - - -")
	fmt.Println(err)
	os.Exit(1)
}

func checkFile(filePath string) error {
	fileStat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("log file at '" + filePath + "' not found")
		}
		return err
	}
	if fileStat.IsDir() {
		return errors.New("log file at '" + filePath + "' is a directory")
	}
	return nil
}

func getServerLogs(filePath string) []string {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		printError(err)
		return []string{"Error while reading log file: " + err.Error()}
	}
	return strings.Split(string(fileContent), "\n")
}
