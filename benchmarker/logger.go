package main

import (
	"log"
	"os"
)

var (
	ContestantLogger = log.New(os.Stdout, "", log.Ltime|log.Lmicroseconds)
	AdminLogger      = log.New(os.Stderr, "[ADMIN] ", log.Ltime|log.Lmicroseconds)
)

func PrintScenarioStarted(scenarioName string) {
	AdminLogger.Printf("シナリオ走行を開始しました [シナリオ名: %s]", scenarioName)
}

func PrintScenarioFinished(scenarioName string) {
	AdminLogger.Printf("シナリオ走行を終了しました [シナリオ名: %s]", scenarioName)
}
